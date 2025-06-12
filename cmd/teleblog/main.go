package main

import (
	"context"
	"fmt"
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
	"gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

func main() {
	config, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	gctx, cancel := context.WithCancel(context.Background())

	// # Pocketbase
	app := pocketbase.New()

	// # Migrations
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	curPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: isGoRun,
		Dir:         path.Join(curPath, "pb_migrations"),
	})

	// # API
	httpapi.InitApi(httpapi.Config{
		Env: config.Env,
	}, app, gctx)

	// # Initialize update history
	admin.InitUploadHistoryUI(app)

	// # Init additional commands
	AdditionalCommands(app)

	// # Bot
	if !config.DisableBot {
		pref := telebot.Settings{
			Verbose: true,
			Token:   config.TelegramBotToken,
			Poller:  &telebot.LongPoller{Timeout: 60 * time.Second, AllowedUpdates: telebot.AllowedUpdates},
			OnError: func(err error, c telebot.Context) {
				app.Logger().Error("Error in bot", "error:", err)
				fmt.Println("Error in bot:", err)
			},
			Synchronous: true,
		}

		b, err := telebot.NewBot(pref)
		if err != nil {
			log.Fatal(
				fmt.Errorf("New bot create error: %s", err),
			)
			return
		}

		b.Use(middleware.Logger())

		botapi.InitBotCommands(b, app)

		go b.Start()
	}

	// # Pre start
	// ## Extract slugs and set album id for posts
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// # Set slug for posts
		if err := features.ExtractSlugs(app); err != nil {
			return fmt.Errorf("Extract slugs error: %w", err)
		}

		// # Set album id for posts
		return features.SetAlbumId(app)
	})

	// ## Prepare DB
	preSeedDB(app)

	// ## Send verification email on sign-up
	app.OnRecordAfterCreateRequest("users").Add(func(e *core.RecordCreateEvent) error {
		return mails.SendRecordVerification(app, e.Record)
	})

	app.Logger().Info("Starting PocketBase server...")

	// # Start app
	if err := app.Start(); err != nil {
		cancel()
		log.Fatal(
			fmt.Errorf("PocketBase start error: %s", err),
		)
	}
}
