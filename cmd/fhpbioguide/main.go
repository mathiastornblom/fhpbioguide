package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"fhpbioguide/pkg/api/bioguide"
	"fhpbioguide/pkg/api/d365"
	"fhpbioguide/pkg/api/handler"
	"fhpbioguide/pkg/logger"
	"fhpbioguide/pkg/repository"
	"fhpbioguide/pkg/syncstate"
	"fhpbioguide/pkg/usecase/cashreports"
	"fhpbioguide/pkg/usecase/movieexport"
	"fhpbioguide/pkg/usecase/theatreexport"

	"github.com/go-co-op/gocron"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
)

var appLog *slog.Logger

func main() {
	initConfig()

	cfg := logger.Config{
		Verbose:    viper.GetBool("log.verbose"),
		File:       viper.GetString("log.file"),
		MaxSizeMB:  viper.GetInt("log.maxSizeMB"),
		MaxBackups: viper.GetInt("log.maxBackups"),
		MaxAgeDays: viper.GetInt("log.maxAgeDays"),
	}
	appLog = logger.New(cfg)

	appLog.Info("starting", "app", "fhpbioguide")
	appLog.Info("log output", "file", cfg.File, "maxSizeMB", cfg.MaxSizeMB, "maxBackups", cfg.MaxBackups, "maxAgeDays", cfg.MaxAgeDays, "verbose", cfg.Verbose)

	dynamicsClient := createDynamicsClient()
	bioguidenClient := createBioguideClient()

	startTriggerPoller(dynamicsClient, bioguidenClient)
	scheduleExportJob(dynamicsClient, bioguidenClient)
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	viper.SetDefault("sync.triggerFile", "/tmp/fhp_sync_trigger")
	viper.SetDefault("sync.lockFile", "/tmp/fhp_sync.lock")
	viper.SetDefault("log.verbose", false)
	viper.SetDefault("log.file", "fhpbioguide.log")
	viper.SetDefault("log.maxSizeMB", 50)
	viper.SetDefault("log.maxBackups", 5)
	viper.SetDefault("log.maxAgeDays", 30)
}

func createDynamicsClient() *d365.D365 {
	return &d365.D365{
		Resty:        resty.New().SetTimeout(30 * time.Second),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
		Logger:       appLog.With("component", "D365"),
	}
}

func createBioguideClient() *bioguide.BioGuiden {
	return &bioguide.BioGuiden{
		Resty:    resty.New().SetTimeout(60 * time.Second),
		URL:      viper.GetViper().GetString("bio.url"),
		Username: viper.GetViper().GetString("bio.username"),
		Password: viper.GetViper().GetString("bio.password"),
		Logger:   appLog.With("component", "BioGuide"),
	}
}

func runSync(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	lockFile := viper.GetString("sync.lockFile")
	l := appLog.With("component", "Sync")

	acquired, err := syncstate.AcquireLock(lockFile, l)
	if err != nil {
		l.Error("lock error", "err", err)
		return
	}
	if !acquired {
		l.Warn("another sync is already running (lock held), skipping")
		return
	}
	l.Info("lock acquired")
	defer func() {
		syncstate.ReleaseLock(lockFile, l)
		l.Info("lock released")
	}()

	jobStart := time.Now()
	lastSync := syncstate.ReadState(l)
	l.Info("starting", "window_from", lastSync.Format("2006-01-02T15:04:05"))

	if err := dynamicsClient.AuthenticateApi(); err != nil {
		l.Error("D365 auth failed", "err", err)
		return
	}

	movieRepository := repository.NewMovieExportRepository(dynamicsClient, bioguidenClient)
	movieService := movieexport.NewService(movieRepository)

	theatreRepository := repository.NewTheatreExportRepository(dynamicsClient, bioguidenClient)
	theatreService := theatreexport.NewService(theatreRepository)

	cashReportRepo := repository.NewCashReportRepository(dynamicsClient, bioguidenClient, appLog.With("component", "CashReportRepo"))
	cashReportService := cashreports.NewService(cashReportRepo)

	if err := handler.ExecuteExports(lastSync, movieService, cashReportService, theatreService, appLog); err != nil {
		l.Error("export failed — state not updated, will retry next run", "err", err)
		return
	}

	if err := syncstate.WriteState(jobStart); err != nil {
		l.Error("could not write sync state", "err", err)
	} else {
		l.Info("completed", "duration", time.Since(jobStart).Round(time.Second).String(), "state_updated_to", jobStart.Format("2006-01-02T15:04:05"))
	}
}

func startTriggerPoller(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	triggerFile := viper.GetString("sync.triggerFile")
	l := appLog.With("component", "Scheduler")
	l.Info("trigger poller started", "path", triggerFile, "interval", "60s")

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if syncstate.TriggerPending(triggerFile) {
				l.Info("trigger file detected — starting on-demand sync")
				syncstate.ClearTrigger(triggerFile)
				runSync(dynamicsClient, bioguidenClient)
			} else {
				appLog.Debug("trigger poll — no trigger pending", "component", "Scheduler")
			}
		}
	}()
}

func scheduleExportJob(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	l := appLog.With("component", "Scheduler")
	s := gocron.NewScheduler(time.Local)
	job, _ := s.Every(1).Days().At("02:00").Do(func() {
		l.Info("scheduled sync triggered")
		runSync(dynamicsClient, bioguidenClient)
	})

	nextRun := job.NextRun()
	if nextRun != (time.Time{}) {
		l.Info("job registered", "schedule", "daily 02:00", "next_run", nextRun.Format("2006-01-02T15:04:05"))
	}

	s.StartBlocking()
	l.Error("scheduler stopped unexpectedly")
	os.Exit(1)
}
