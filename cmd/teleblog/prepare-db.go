package main

import (
	"fmt"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
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
		return fmt.Errorf("Query existing tags error: %w", err)
	}

	if len(existingTags) == 0 {
		err = features.ExtractAndSaveAllTags(app)
		if err != nil {
			return fmt.Errorf("Extract and save all tags error: %w", err)
		}
	}

	// # Fix comments without posta
	commentsWithoutPost := make([]teleblog.Comment, 0)
	err = teleblog.CommentQuery(app.Dao()).
		Where(
			dbx.NewExp(
				`post_id = ""`,
			),
		).
		AndWhere(
			dbx.NewExp(
				"tg_reply_to_message_id > 0",
			),
		).
		AndWhere(
			dbx.NewExp(
				"is_tg_history_message = false",
			),
		).
		All(&commentsWithoutPost)
	if err != nil {
		return fmt.Errorf("Query comments without post error: %w", err)
	}

	app.Logger().Warn("Found comments without post", "count", len(commentsWithoutPost))

	// TODO: Uncomment this block when ready to fix comments without posts
	// for _, comment := range commentsWithoutPost {
	// 	jb, err := comment.TgMessageRaw.MarshalJSON()
	// 	if err != nil {
	// 		app.Logger().Error("PrepareDB: marshal comment error", "error", err)
	// 		continue
	// 	}

	// 	rawMessage := telebot.Message{}

	// 	err = json.Unmarshal(jb, &rawMessage)
	// 	if err != nil {
	// 		app.Logger().Error("PrepareDB: unmarshal comment error", "error", err)
	// 		continue
	// 	}

	// 	if rawMessage.ReplyTo != nil && rawMessage.ReplyTo.OriginalMessageID > 0 {
	// 		// Find the post by tg_reply_to_message_id
	// 		post := teleblog.Post{}

	// 		err := teleblog.PostQuery(app.Dao()).
	// 			Where(dbx.HashExp{"tg_message_id": rawMessage.ReplyTo.OriginalMessageID}).
	// 			One(&post)
	// 		if err != nil {
	// 			if strings.Contains(err.Error(), "no rows in result set") {
	// 				continue // Post not found, skip this comment
	// 			}
	// 			return fmt.Errorf("Query post by tg_reply_to_message_id error: %w", err)
	// 		}

	// 		comment.PostId = post.Id
	// 		if err := app.Dao().Save(&comment); err != nil {
	// 			return fmt.Errorf("Failed to update comment with post ID: %w", err)
	// 		}

	// 		post.TgGroupMessageId = rawMessage.ThreadID
	// 		if err := app.Dao().Save(&post); err != nil {
	// 			return fmt.Errorf("Failed to update post with tg_group_message_id: %w", err)
	// 		}

	// 		app.Logger().Info("Updated comment with post ID", "comment_id", comment.Id, "post_id", post.Id)
	// 	}
	// }

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
