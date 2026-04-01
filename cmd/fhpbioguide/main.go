package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"fhpbioguide/pkg/api/bioguide"          // Package for handling the BioGuiden API
	"fhpbioguide/pkg/api/d365"              // Package for handling the Dynamics 365 API
	"fhpbioguide/pkg/api/handler"           // Package for executing data export tasks
	"fhpbioguide/pkg/repository"            // Package containing data repositories for different entities
	"fhpbioguide/pkg/syncstate"             // Package for persisting sync state across runs
	"fhpbioguide/pkg/usecase/cashreports"   // Package for handling cash reports use case
	"fhpbioguide/pkg/usecase/movieexport"   // Package for handling movie exports use case
	"fhpbioguide/pkg/usecase/theatreexport" // Package for handling theatre exports use case

	"github.com/go-co-op/gocron"   // Package for scheduling tasks
	"github.com/go-resty/resty/v2" // Package for making RESTful HTTP requests
	"github.com/spf13/viper"       // Package for managing application configurations
)

// main is the entry point of the application
func main() {
	initConfig()

	// Open or create the log file for storing logs
	file, err := os.OpenFile("fhpbioguide.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}

	dynamicsClient := createDynamicsClient()
	bioguidenClient := createBioguideClient(file)

	// Start the trigger file poller — checks every 60s for a file written by fhpreports.
	// Must be started before the blocking scheduler.
	startTriggerPoller(dynamicsClient, bioguidenClient)

	// Schedule the daily export job and block.
	scheduleExportJob(dynamicsClient, bioguidenClient)
}

// initConfig initializes the configuration using Viper
func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// Defaults for trigger/lock paths so both apps work without explicit config.
	viper.SetDefault("sync.triggerFile", "/tmp/fhp_sync_trigger")
	viper.SetDefault("sync.lockFile", "/tmp/fhp_sync.lock")
}

// createDynamicsClient sets up the Dynamics 365 API client with configuration values
func createDynamicsClient() *d365.D365 {
	return &d365.D365{
		Resty:        resty.New().SetTimeout(30 * time.Second),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
	}
}

// createBioguideClient sets up the BioGuiden API client with configuration values and a log file
func createBioguideClient(file *os.File) *bioguide.BioGuiden {
	return &bioguide.BioGuiden{
		Resty:    resty.New().SetTimeout(60 * time.Second),
		URL:      viper.GetViper().GetString("bio.url"),
		Username: viper.GetViper().GetString("bio.username"),
		Password: viper.GetViper().GetString("bio.password"),
		Logger:   log.New(file, "PRODUCTION: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// runSync executes a full sync cycle guarded by a cross-process lock file.
// Both the scheduled job and the trigger poller call this function, so only
// one sync can run at a time regardless of how it was initiated.
func runSync(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	lockFile := viper.GetString("sync.lockFile")

	acquired, err := syncstate.AcquireLock(lockFile)
	if err != nil {
		log.Printf("sync: lock error: %v", err)
		return
	}
	if !acquired {
		log.Printf("sync: another sync is already running (lock held), skipping")
		return
	}
	defer syncstate.ReleaseLock(lockFile)

	jobStart := time.Now()
	lastSync := syncstate.ReadState()
	log.Printf("sync: starting — window %s → now", lastSync.Format("2006-01-02T15:04:05"))

	// Reauthenticate — token auto-refresh also handles mid-run expiry.
	if err := dynamicsClient.AuthenticateApi(); err != nil {
		log.Printf("sync: D365 auth error: %v", err)
	}

	movieRepository := repository.NewMovieExportRepository(dynamicsClient, bioguidenClient)
	movieService := movieexport.NewService(movieRepository)

	theatreRepository := repository.NewTheatreExportRepository(dynamicsClient, bioguidenClient)
	theatreService := theatreexport.NewService(theatreRepository)

	cashReportRepo := repository.NewCashReportRepository(dynamicsClient, bioguidenClient)
	cashReportService := cashreports.NewService(cashReportRepo)

	if err := handler.ExecuteExports(lastSync, movieService, cashReportService, theatreService); err != nil {
		log.Printf("sync: export error: %v — state not updated, will retry next run", err)
		return
	}

	if err := syncstate.WriteState(jobStart); err != nil {
		log.Printf("sync: could not write state: %v", err)
	} else {
		log.Printf("sync: completed, state updated to %s", jobStart.Format("2006-01-02T15:04:05"))
	}
}

// startTriggerPoller starts a goroutine that checks for a trigger file every 60 seconds.
// When fhpreports writes the trigger file (via POST /api/sync/trigger), this poller
// picks it up, deletes it, and calls runSync. The lock inside runSync prevents overlap
// with the scheduled job.
func startTriggerPoller(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	triggerFile := viper.GetString("sync.triggerFile")
	log.Printf("sync: trigger poller started (watching %s every 60s)", triggerFile)

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if syncstate.TriggerPending(triggerFile) {
				log.Printf("sync: trigger file detected, consuming and starting sync")
				syncstate.ClearTrigger(triggerFile)
				runSync(dynamicsClient, bioguidenClient)
			}
		}
	}()
}

// scheduleExportJob schedules a daily sync at 02:00 and blocks.
func scheduleExportJob(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	s := gocron.NewScheduler(time.Local)
	s.Every(1).Days().At("02:00").Do(func() {
		fmt.Printf("Scheduled sync at 02:00\n\r")
		log.Printf("Scheduled sync at 02:00\n\r")
		runSync(dynamicsClient, bioguidenClient)
	})
	s.StartBlocking()
}
