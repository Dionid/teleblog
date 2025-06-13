package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/file"
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
					log.Fatal("recover: ", r)
				}
			})()

			fileName := "export.zip"

			if len(args) > 0 {
				fileName = args[0]
			}

			// Check if the file is a zip file
			if ext := filepath.Ext(fileName); ext != ".zip" {
				log.Fatal(errors.New("Not zip"))
			}

			// Read the file content
			reader, err := os.Open(fileName)
			if err != nil {
				log.Fatal(
					fmt.Errorf("Failed to open file %s: %w", fileName, err),
				)
			}
			defer reader.Close()

			// Read the entire file into memory
			fileBytes, err := io.ReadAll(reader)
			if err != nil {
				log.Fatal(
					fmt.Errorf("Failed to read file %s: %w", fileName, err),
				)
			}

			// Create a bytes reader which implements io.ReaderAt
			zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
			if err != nil {
				log.Fatal(
					fmt.Errorf("Failed to create zip reader for file %s: %w", fileName, err),
				)
			}

			// Unzip the zip file
			folderPathPrefix := "extracted-" + time.Now().Format("20060102150405")
			err = file.Unzip(zipReader, folderPathPrefix)
			if err != nil {
				log.Fatal(
					fmt.Errorf("Failed to unzip file %s: %w", fileName, err),
				)
			}
			defer os.RemoveAll(folderPathPrefix)

			err = features.UploadHistory(app, folderPathPrefix)
			if err != nil {
				log.Fatal(
					fmt.Errorf("Failed to upload history from file %s: %w", fileName, err),
				)
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
