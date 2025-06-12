package httpapi

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/Dionid/teleblog/libs/file"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Config struct {
	Env string
}

func CacheControlMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "public, max-age=86400")
		return next(c)
	}
}

//go:embed public
var publicAssets embed.FS

func InitApi(config Config, app core.App, gctx context.Context) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.Use(apis.ActivityLogger(app))

		// # Static
		if config.Env == "PRODUCTION" {
			os.RemoveAll("./public")
			file.CopyFromEmbed(publicAssets, "public", "./public")
			subFs := echo.MustSubFS(e.Router.Filesystem, "./public")
			e.Router.Add(
				http.MethodGet,
				"/public"+"*",
				echo.StaticDirectoryHandler(subFs, false),
				CacheControlMiddleware,
			)
		} else if config.Env == "LOCAL" {
			subFs := echo.MustSubFS(e.Router.Filesystem, "./httpapi/public")
			e.Router.Add(
				http.MethodGet,
				"/public"+"*",
				echo.StaticDirectoryHandler(subFs, false),
				CacheControlMiddleware,
			)
		} else {
			log.Fatalf("Unknown env: %s", config.Env)
		}

		IndexPageHandler(config, e, app)
		SiteMapAndRobotsPageHandler(e, app)
		PostPageHandler(e, app)

		return nil
	})
}
