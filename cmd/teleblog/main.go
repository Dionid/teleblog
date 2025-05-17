package main

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Dionid/teleblog/cmd/teleblog/admin"
	"github.com/Dionid/teleblog/cmd/teleblog/botapi"
	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/cmd/teleblog/httpapi"
	_ "github.com/Dionid/teleblog/cmd/teleblog/pb_migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/mails"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"gopkg.in/telebot.v3"
)

func main() {
	config, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	gctx, _ := context.WithCancel(context.Background())

	// # Pocketbase
	app := pocketbase.New()

	// # Send verification email on sign-up
	app.OnRecordAfterCreateRequest("users").Add(func(e *core.RecordCreateEvent) error {
		return mails.SendRecordVerification(app, e.Record)
	})

	// Initialize admin UI with custom buttons
	admin.InitUploadHistoryUI(app)

	AdditionalCommands(app)

	// # Migrations
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	curPath, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: isGoRun,
		Dir:         path.Join(curPath, "pb_migrations"),
	})

	// # HTTP API
	httpapi.InitApi(httpapi.Config{
		Env:    config.Env,
		UserId: config.UserId,
	}, app, gctx)

	preSeedDB(app)

	// # Bot
	if !config.DisableBot {
		pref := telebot.Settings{
			Verbose: false,
			Token:   config.TelegramBotToken,
			Poller:  &telebot.LongPoller{Timeout: 60 * time.Second, AllowedUpdates: telebot.AllowedUpdates},
			OnError: func(err error, c telebot.Context) {
				app.Logger().Error("Error in bot", "error:", err)
			},
			Synchronous: true,
		}

		b, err := telebot.NewBot(pref)
		if err != nil {
			log.Fatal(err)
			return
		}

		botapi.InitBotCommands(b, app)

		go b.Start()
	}

	// # Pre start
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// # Set albumId for posts
		if err := features.ExtractSlugs(app); err != nil {
			return err
		}

		return features.SetAlbumId(app)
	})

	// # Start app
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
