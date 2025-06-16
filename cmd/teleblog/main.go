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
	"github.com/Dionid/teleblog/cmd/teleblog/httpapi"
	_ "github.com/Dionid/teleblog/cmd/teleblog/pb_migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"gopkg.in/telebot.v4"
)

func main() {
	// # Context
	gctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// # Config
	config, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	// # Pocketbase
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())
	isServerCmd := false

	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "serve") {
			isServerCmd = true
			break
		}
	}

	app := pocketbase.NewWithConfig(
		pocketbase.Config{
			DefaultDev: !isServerCmd || isGoRun,
		},
	)

	// # Migrations

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

	// # Init additional commands
	AdditionalCommands(app)

	// # Init
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		app.Logger().Info("Starting PocketBase server...")

		// # Initialize update history
		err := admin.InitUploadHistoryUI(app, e)
		if err != nil {
			return fmt.Errorf("failed to initialize upload history UI: %w", err)
		}

		// # Bot
		if !config.DisableBot {
			pref := telebot.Settings{
				Verbose: config.TelegramBotVerbose,
				Token:   config.TelegramBotToken,
				Poller:  &telebot.LongPoller{Timeout: 60 * time.Second, AllowedUpdates: telebot.AllowedUpdates},
				OnError: func(err error, c telebot.Context) {
					app.Logger().Error("Error in bot", "error:", err)
				},
				Synchronous: true,
			}

			b, err := telebot.NewBot(pref)
			if err != nil {
				return fmt.Errorf("failed to create bot: %w", err)
			}

			err = botapi.InitBotCommands(b, app)
			if err != nil && !strings.Contains(err.Error(), "retry after") {
				return fmt.Errorf("Init bot commands error: %s", err)
			}

			go b.Start()
		}

		// # Prepare DB
		if !config.DisablePrepareDB {
			err := prepareDB(app)
			if err != nil {
				return fmt.Errorf("prepare DB error: %s", err)
			}
		}

		return nil
	})

	// # Start app
	if err := app.Start(); err != nil {
		log.Fatal(
			fmt.Errorf("PocketBase start error: %s", err),
		)
	}
}
