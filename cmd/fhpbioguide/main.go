package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"fhpbioguide/pkg/api/bioguide"
	"fhpbioguide/pkg/api/d365"
	"fhpbioguide/pkg/api/handler"
	"fhpbioguide/pkg/repository"
	"fhpbioguide/pkg/usecase/cashreports"
	"fhpbioguide/pkg/usecase/movieexport"
	"fhpbioguide/pkg/usecase/theatreexport"

	"github.com/go-co-op/gocron"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
)

func main() {
	// Read config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	file, _ := os.OpenFile("fhpbioguide.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	// Setup dynamicsClient with data from config
	dynamicsClient := &d365.D365{
		Resty:        resty.New(),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
	}

	// Setup bioguidenClient with data from config
	bioguidenClient := &bioguide.BioGuiden{
		Resty:    resty.New(),
		URL:      viper.GetViper().GetString("bio.url"),
		Username: viper.GetViper().GetString("bio.username"),
		Password: viper.GetViper().GetString("bio.password"),
		Logger:   log.New(file, "PRODUCTION: ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	// Run export job at 02:00 every day
	s := gocron.NewScheduler(time.Local)
	s.Every(1).Days().At("02:00").Do(func() {
		// Reauthenticate api token for dynamicsClient
		dynamicsClient.AuthenticateApi()

		movieRepository := repository.NewMovieExportRepository(dynamicsClient, bioguidenClient)
		movieService := movieexport.NewService(movieRepository)

		theatreRepository := repository.NewTheatreExportRepository(dynamicsClient, bioguidenClient)
		theatreService := theatreexport.NewService(theatreRepository)

		cashReportRepo := repository.NewCashReportRepository(dynamicsClient, bioguidenClient)
		cashReportService := cashreports.NewService(cashReportRepo)
		handler.ExecuteExports(movieService, cashReportService, theatreService)
	})

	s.StartBlocking()
}
