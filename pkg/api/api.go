package api

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
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
	"github.com/gofiber/template/html/v2"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type API struct {
	App               *fiber.App
	ReportFormService *reportform.Service
	log               *slog.Logger
}

func NewAPI(log *slog.Logger) *API {
	engine := html.New("./views", ".html")
	return &API{
		log: log,
		App: fiber.New(fiber.Config{
			Views: engine,
			ErrorHandler: func(ctx *fiber.Ctx, err error) error {
				code := fiber.StatusInternalServerError
				if e, ok := err.(*fiber.Error); ok {
					code = e.Code
				}
				err = ctx.Status(code).Render("error", fiber.Map{
					"Title": "Fel",
					"Desc":  "Länken är felaktig eller använd. Kontakta Folketshus och Parker för support.",
					"Error": strconv.Itoa(code),
				})
				if err != nil {
					return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
				}
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
	db.AutoMigrate(entity.Form{}, entity.Event{})
	a.App.Use(cors.New())
	a.App.Use(recover.New())
	a.App.Use(compress.New())
	a.App.Static("/css", "./views/css")
	a.App.Static("/script", "./views/script")
	a.App.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	reportformRepo := repository.NewReportFormRepository(db, a.log.With("component", "D365"))
	a.ReportFormService = reportform.NewService(reportformRepo)

	handler.MakeReportForms(a.App, a.ReportFormService, a.log)

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(viper.GetString("report.url")),
		Cache:      autocert.DirCache("./certs"),
	}

	cfg := &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"http/1.1", "acme-tls/1"},
	}

	ln, err := tls.Listen("tcp", ":443", cfg)
	if err != nil {
		a.log.Error("failed to bind TLS listener", "err", err)
		os.Exit(1)
	}

	go http.ListenAndServe(":80", http.HandlerFunc(a.redirect))

	a.log.Info("server listening", "addr", ":443")
	if err := a.App.Listener(ln); err != nil {
		a.log.Error("server stopped unexpectedly", "err", err)
		os.Exit(1)
	}
}

func (a *API) redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	a.log.Debug("HTTP→HTTPS redirect", "target", target)
	http.Redirect(w, req, target, http.StatusMovedPermanently)
}
