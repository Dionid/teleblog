package main

import (
	"fmt"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/security"
	"golang.org/x/crypto/bcrypt"
)

func prepareDB(app *pocketbase.PocketBase, config *Config) error {
	// # Set slug for posts
	if err := features.ExtractSlugs(app); err != nil {
		return fmt.Errorf("Extract slugs error: %w", err)
	}

	// # Set album id for posts
	err := features.SetAlbumId(app)
	if err != nil {
		return fmt.Errorf("Set album id error: %w", err)
	}

	var existingTags []teleblog.Tag
	err = teleblog.TagQuery(app.Dao()).
		Limit(1).
		All(&existingTags)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil
		}
		return fmt.Errorf("Query existing tags error: %w", err)
	}

	if len(existingTags) > 0 {
		return nil
	}

	err = features.ExtractAndSaveAllTags(app)
	if err != nil {
		return fmt.Errorf("Extract and save all tags error: %w", err)
	}

	// # Prepare users
	user := teleblog.User{}
	err = teleblog.UserQuery(app.Dao()).
		Limit(1).
		One(&user)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			return err
		}

		// hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(
			config.TelegramBotToken[0:9],
		), 12)
		if err != nil {
			return err
		}

		user = teleblog.User{
			Username:     "default",
			PasswordHash: string(hashedPassword),
			TokenKey:     security.RandomString(50),
		}
		if err := app.Dao().Save(&user); err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}

		tgToken := teleblog.TgVerificationToken{
			UserId:   user.Id,
			Value:    uuid.New().String(),
			Verified: false,
		}
		if err := app.Dao().Save(&tgToken); err != nil {
			return fmt.Errorf("failed to create default verification token: %w", err)
		}

		app.Logger().Info("Created default user and verification token", "user_id", user.Id, "token", tgToken.Value)
	}

	return nil
}
