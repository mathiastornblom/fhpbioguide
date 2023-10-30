package api

import (
	"crypto/tls"
	"log"
	"net/http"
	"strconv"

	"fhpbioguide/pkg/api/handler"
	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/repository"
	"fhpbioguide/pkg/usecase/reportform"

	"golang.org/x/crypto/acme/autocert"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type API struct {
	App               *fiber.App
	ReportFormService *reportform.Service
}

func NewAPI() *API {
	engine := html.New("./views", ".html")
	return &API{
		App: fiber.New(fiber.Config{
			Views: engine,
			// Override default error handler
			ErrorHandler: func(ctx *fiber.Ctx, err error) error {
				// Status code defaults to 500
				code := fiber.StatusInternalServerError

				// Retrieve the custom status code if it's an fiber.*Error
				if e, ok := err.(*fiber.Error); ok {
					code = e.Code
				}

				// Send custom error page
				err = ctx.Status(code).Render("error", fiber.Map{
					"Title": "Fel",
					"Desc":  "Länken är felaktig eller använd. Kontakta Folketshus och Parker för support.",
					"Error": strconv.Itoa(code),
				})

				if err != nil {
					// In case the SendFile fails
					return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
				}

				// Return from handler
				return nil
			},
		}),
	}
}

func (a *API) StartAPI() {

	dsn := "root:root@tcp(127.0.0.1:3306)/fhpreports?charset=utf8&parseTime=True"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(entity.Form{}, entity.Event{}) // database auto migrate
	a.App.Use(cors.New())                         // cors module
	a.App.Use(recover.New())                      // Auto recover module

	// Default middleware config
	a.App.Use(compress.New())

	// Serve static css files
	a.App.Static("css", "./views/css")

	// Provide a custom compression level
	a.App.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	reportformRepo := repository.NewReportFormRepository(db)
	a.ReportFormService = reportform.NewService(reportformRepo)

	handler.MakeReportForms(a.App, a.ReportFormService)

	// Certificate manager
	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(viper.GetString("report.url")),
		// Folder to store the certificates
		Cache: autocert.DirCache("./certs"),
	}

	// TLS Config
	cfg := &tls.Config{
		// Get Certificate from Let's Encrypt
		GetCertificate: m.GetCertificate,
		// By default NextProtos contains the "h2"
		// This has to be removed since Fasthttp does not support HTTP/2
		// Or it will cause a flood of PRI method logs
		// http://webconcepts.info/concepts/http-method/PRI
		NextProtos: []string{
			"http/1.1", "acme-tls/1",
		},
	}

	ln, err := tls.Listen("tcp", ":443", cfg)
	if err != nil {
		panic(err)
	}
	// start standard http server on port 80 to handle auto ssl redirect
	go http.ListenAndServe(":80", http.HandlerFunc(redirect))

	log.Fatal(a.App.Listener(ln))

	// a.App.Listen(":1480")
}

func redirect(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := "https://" + req.Host + req.URL.Path
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusMovedPermanently)
}
