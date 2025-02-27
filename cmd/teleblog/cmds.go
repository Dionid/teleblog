package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/pocketbase"
	"github.com/spf13/cobra"
)

func AdditionalCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "reset-password",
		Short: "Reset admin password",
		Long:  "Reset the password for an admin user. Usage: reset-password [email] [new-password]",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			email := args[0]
			newPassword := args[1]

			admin, err := app.Dao().FindAdminByEmail(email)
			if err != nil {
				log.Fatalf("Failed to find admin with email %s: %v", email, err)
			}

			if err := admin.SetPassword(newPassword); err != nil {
				log.Fatalf("Failed to set new password: %v", err)
			}

			if err := app.Dao().SaveAdmin(admin); err != nil {
				log.Fatalf("Failed to save admin: %v", err)
			}

			fmt.Printf("Successfully reset password for admin %s\n", email)
		},
	})

	app.RootCmd.AddCommand(&cobra.Command{
		Use: "upload-history",
		Run: func(cmd *cobra.Command, args []string) {
			defer (func() {
				if r := recover(); r != nil {
					log.Fatal("recover", r)
				}
			})()

			fileName := "result.json"

			if len(args) > 0 {
				fileName = args[0]
			}

			file, err := os.ReadFile(fileName)
			if err != nil {
				log.Fatal(err)
			}

			var history teleblog.History
			err = json.Unmarshal(file, &history)
			if err != nil {
				log.Fatal(err)
			}

			err = features.UploadHistory(app, history)
			if err != nil {
				log.Fatal(err)
			}

			app.Logger().Info("Done")
		},
	})

	app.RootCmd.AddCommand(&cobra.Command{
		Use: "extract-tags",
		Run: func(cmd *cobra.Command, args []string) {
			defer (func() {
				if r := recover(); r != nil {
					log.Fatal("recover", r)
				}
			})()

			err := features.ExtractAndSaveAllTags(app)
			if err != nil {
				log.Fatal(err)
			}

			app.Logger().Info("Done")
		},
	})
}
