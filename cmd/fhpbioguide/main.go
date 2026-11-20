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

// cashAccount is one BioGuiden distributor login used to fetch cash reports.
type cashAccount struct {
	name   string
	id     int // written to new_distributorid in D365
	client *bioguide.BioGuiden
}

// extraDistributor is an entry in bio.extraCashDistributors in config.yaml.
type extraDistributor struct {
	Name     string `mapstructure:"name"`
	ID       int    `mapstructure:"id"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

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
	cashAccounts := createCashAccounts(bioguidenClient)
	for _, a := range cashAccounts {
		appLog.Info("cash report account configured", "name", a.name, "distributor_id", a.id)
	}

	startTriggerPoller(dynamicsClient, bioguidenClient, cashAccounts)
	scheduleExportJob(dynamicsClient, bioguidenClient, cashAccounts)
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	viper.SetDefault("bio.id", 26) // FHP Riks
	viper.SetDefault("bio.name", "FHP Riks")
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

// createCashAccounts returns the accounts cash reports are fetched from: the main
// bio account (FHP Riks) first, then every entry in bio.extraCashDistributors.
// It panics on an incomplete entry, so a misconfiguration is caught at startup
// instead of producing rows without distributor id.
func createCashAccounts(main *bioguide.BioGuiden) []cashAccount {
	accounts := []cashAccount{{
		name:   viper.GetString("bio.name"),
		id:     viper.GetInt("bio.id"),
		client: main,
	}}

	var extras []extraDistributor
	if err := viper.UnmarshalKey("bio.extraCashDistributors", &extras); err != nil {
		panic(fmt.Errorf("invalid bio.extraCashDistributors in config: %w", err))
	}
	seen := map[int]bool{accounts[0].id: true}
	for i, e := range extras {
		if e.ID <= 0 || e.Username == "" || e.Password == "" {
			panic(fmt.Errorf("bio.extraCashDistributors[%d] (%q): id, username and password are required", i, e.Name))
		}
		if seen[e.ID] {
			panic(fmt.Errorf("bio.extraCashDistributors[%d] (%q): duplicate distributor id %d", i, e.Name, e.ID))
		}
		seen[e.ID] = true
		name := e.Name
		if name == "" {
			name = fmt.Sprintf("distributor %d", e.ID)
		}
		accounts = append(accounts, cashAccount{
			name: name,
			id:   e.ID,
			client: &bioguide.BioGuiden{
				Resty:    resty.New().SetTimeout(60 * time.Second),
				URL:      viper.GetString("bio.url"),
				Username: e.Username,
				Password: e.Password,
				Logger:   appLog.With("component", "BioGuide", "distributor", name),
			},
		})
	}
	return accounts
}

func runSync(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden, cashAccounts []cashAccount) {
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

	sources := make([]handler.CashSource, 0, len(cashAccounts))
	for _, a := range cashAccounts {
		repo := repository.NewCashReportRepository(dynamicsClient, a.client, appLog.With("component", "CashReportRepo", "distributor", a.name))
		sources = append(sources, handler.CashSource{Name: a.name, DistributorID: a.id, Service: cashreports.NewService(repo)})
	}

	if err := handler.ExecuteExports(lastSync, movieService, sources, theatreService, appLog); err != nil {
		l.Error("export failed — state not updated, will retry next run", "err", err)
		return
	}

	if err := syncstate.WriteState(jobStart); err != nil {
		l.Error("could not write sync state", "err", err)
	} else {
		l.Info("completed", "duration", time.Since(jobStart).Round(time.Second).String(), "state_updated_to", jobStart.Format("2006-01-02T15:04:05"))
	}
}

func startTriggerPoller(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden, cashAccounts []cashAccount) {
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
				runSync(dynamicsClient, bioguidenClient, cashAccounts)
			} else {
				appLog.Debug("trigger poll — no trigger pending", "component", "Scheduler")
			}
		}
	}()
}

func scheduleExportJob(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden, cashAccounts []cashAccount) {
	l := appLog.With("component", "Scheduler")
	s := gocron.NewScheduler(time.Local)
	job, _ := s.Every(1).Days().At("02:00").Do(func() {
		l.Info("scheduled sync triggered")
		runSync(dynamicsClient, bioguidenClient, cashAccounts)
	})

	nextRun := job.NextRun()
	if nextRun != (time.Time{}) {
		l.Info("job registered", "schedule", "daily 02:00", "next_run", nextRun.Format("2006-01-02T15:04:05"))
	}

	s.StartBlocking()
	l.Error("scheduler stopped unexpectedly")
	os.Exit(1)
}
