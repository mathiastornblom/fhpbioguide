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
	"fhpbioguide/pkg/usecase/cashreports"   // Package for handling cash reports use case
	"fhpbioguide/pkg/usecase/movieexport"   // Package for handling movie exports use case
	"fhpbioguide/pkg/usecase/theatreexport" // Package for handling theatre exports use case

	"github.com/go-co-op/gocron"   // Package for scheduling tasks
	"github.com/go-resty/resty/v2" // Package for making RESTful HTTP requests
	"github.com/spf13/viper"       // Package for managing application configurations
)

// main is the entry point of the application
func main() {
	// Initialize the application configuration
	initConfig()

	// Open or create the log file for storing logs
	file, err := os.OpenFile("fhpbioguide.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}

	// Create the Dynamics 365 and BioGuiden API clients
	dynamicsClient := createDynamicsClient()
	bioguidenClient := createBioguideClient(file)

	// Schedule the daily export job to run at 02:00
	scheduleExportJob(dynamicsClient, bioguidenClient)
}

// initConfig initializes the configuration using Viper
func initConfig() {
	// Read and parse the config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

// createDynamicsClient sets up the Dynamics 365 API client with configuration values
func createDynamicsClient() *d365.D365 {
	return &d365.D365{
		Resty:        resty.New(),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
	}
}

// createBioguideClient sets up the BioGuiden API client with configuration values and a log file
func createBioguideClient(file *os.File) *bioguide.BioGuiden {
	return &bioguide.BioGuiden{
		Resty:    resty.New(),
		URL:      viper.GetViper().GetString("bio.url"),
		Username: viper.GetViper().GetString("bio.username"),
		Password: viper.GetViper().GetString("bio.password"),
		Logger:   log.New(file, "PRODUCTION: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// scheduleExportJob schedules a daily export job to run at 02:00, using the provided API clients
func scheduleExportJob(dynamicsClient *d365.D365, bioguidenClient *bioguide.BioGuiden) {
	// Initialize a new task scheduler
	s := gocron.NewScheduler(time.Local)
	s.Every(1).Days().At("02:00").Do(func() {
		fmt.Printf("Creates a scheduled task at 02:00 \n\r")
		log.Printf("Creates a scheduled task at 02:00 \n\r")
		// Reauthenticate api token for dynamicsClient
		dynamicsClient.AuthenticateApi()

		// Create movie export repository and service
		movieRepository := repository.NewMovieExportRepository(dynamicsClient, bioguidenClient)
		movieService := movieexport.NewService(movieRepository)

		// Create theatre export repository and service
		theatreRepository := repository.NewTheatreExportRepository(dynamicsClient, bioguidenClient)
		theatreService := theatreexport.NewService(theatreRepository)

		// Create cash report repository and service
		cashReportRepo := repository.NewCashReportRepository(dynamicsClient, bioguidenClient)
		cashReportService :=
			cashreports.NewService(cashReportRepo)

		// Execute the data export tasks for movies, cash reports, and theatres
		handler.ExecuteExports(movieService, cashReportService, theatreService)
	})

	// Start the task scheduler and block the main function to keep the scheduler running
	s.StartBlocking()
}
