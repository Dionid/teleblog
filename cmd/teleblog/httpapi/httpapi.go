package httpapi

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/Dionid/teleblog/libs/file"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Config struct {
	Env    string
	UserId string
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
			e.Router.Static("/public", "./public")
		} else if config.Env == "LOCAL" {
			e.Router.Static("/public", "./httpapi/public")
		} else {
			log.Fatalf("Unknown env: %s", config.Env)
		}

		IndexPageHandler(config, e, app)
		SiteMapAndRobotsPageHandler(e, app)
		PostPageHandler(e, app)

		return nil
	})
}
