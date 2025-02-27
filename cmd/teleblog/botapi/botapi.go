package botapi

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
	"gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

const ADD_CHANNEL_COMMAND_NAME = "addchannel"
const VERIFY_TOKEN_COMMAND_NAME = "verifytoken"

func skipContent(_ telebot.Context) bool {
	// # We can't skip content, because we need all posts for links
	return false
}

// downloadPhoto downloads a photo from a message to the specified directory
func downloadPhoto(b *telebot.Bot, photo *telebot.Photo, outputDir string) (string, error) {
	// Get file info from Telegram
	file, err := b.FileByID(photo.FileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Generate unique filename
	filename := filepath.Join(outputDir, fmt.Sprintf("%s.jpg", photo.UniqueID))

	// Download the file
	err = b.Download(&file, filename)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	return filename, nil
}

func InitBotCommands(b *telebot.Bot, app *pocketbase.PocketBase) {
	err := b.SetCommands([]telebot.Command{
		{Text: "start", Description: "start the bot"},
		{Text: VERIFY_TOKEN_COMMAND_NAME, Description: "send token to bind bot to your telebot account (e.g. /verifytoken YOUR_TOKEN)"},
		{Text: ADD_CHANNEL_COMMAND_NAME, Description: "send channel to create blog from it (e.g. /addchannel @YOUR_CHANNEL_NAME)"},
	})
	if err != nil {
		log.Fatal(err)
	}

	b.Handle("/start", func(c telebot.Context) error {
		return c.Reply("Hello! This is teleblog bot. Add it to your channel and get posts in your blog.")
	})

	b.Use(middleware.Recover(func(err error, ctx telebot.Context) {
		app.Logger().Error("Error in bot", err)
	}))

	VerifyTokenCommand(b, app)
	AddChannelCommand(b, app)

	b.Handle(telebot.OnChannelPost, func(c telebot.Context) error {
		chat := &teleblog.Chat{}

		if skipContent(c) {
			return nil
		}

		err = teleblog.ChatQuery(app.Dao()).
			AndWhere(dbx.HashExp{"tg_chat_id": c.Chat().ID}).
			Limit(1).
			One(chat)
		if err != nil {
			return err
		}

		newPost := &teleblog.Post{
			ChatId:      chat.Id,
			IsTgMessage: true,
			Text:        c.Message().Text + c.Message().Caption,
			TgMessageId: c.Message().ID,
		}

		newPost.Created.Scan(c.Message().Time())

		jsonMessageRaw, err := json.Marshal(c.Message())
		if err != nil {
			return err
		}

		err = newPost.TgMessageRaw.Scan(jsonMessageRaw)
		if err != nil {
			return err
		}

		err = app.Dao().Save(newPost)
		if err != nil {
			return err
		}

		// Handle photo if present
		if photo := c.Message().Photo; photo != nil {
			postCollection, err := app.Dao().FindCollectionByNameOrId("post")
			if err != nil {
				return err
			}

			fsys, err := app.NewFilesystem()
			if err != nil {
				return err
			}
			defer fsys.Close()

			outputDir := "temp-tg-webhook-uploads-" + newPost.Id
			// Create output directory if it doesn't exist
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			defer os.RemoveAll(outputDir)

			filename, err := downloadPhoto(b, photo, outputDir)
			if err != nil {
				return fmt.Errorf("failed to download photo: %w", err)
			}

			file, err := filesystem.NewFileFromPath(filename)
			if err != nil {
				return err
			}

			fileName := postCollection.Id + "/" + newPost.Id + "/" + file.Name

			fmt.Println("fileName: ", fileName)

			err = fsys.UploadFile(file, fileName)
			if err != nil {
				return err
			}

			newPost.Photos = append(newPost.Photos, file.Name)
		}

		err = app.Dao().Save(newPost)
		if err != nil {
			return err
		}

		return nil
	})

	// # Created messages in channels, groups and bot
	b.Handle(telebot.OnText, func(c telebot.Context) error {
		app.Logger().Info("telebot.OnText")

		if skipContent(c) {
			return nil
		}

		// # 0 if reply to something, or Post.Id if reply to post
		if c.Message().ReplyTo != nil {
			fmt.Println("c.Message().ReplyTo.OriginalMessageID", c.Message().ReplyTo.OriginalMessageID)
		}

		chat := &teleblog.Chat{}
		err = teleblog.ChatQuery(app.Dao()).
			AndWhere(dbx.HashExp{"tg_chat_id": c.Chat().ID}).
			Limit(1).
			One(chat)
		if err != nil {
			return err
		}

		if c.Message().FromChannel() {
			fmt.Println("Channel!", c.Message().Text)
		} else if c.Message().FromGroup() {
			fmt.Println("Group!", c.Message().Text)

			// # Forward from channel to group
			if c.Message().OriginalChat != nil && c.Message().OriginalChat.ID == chat.TgLinkedChatId {
				_, err := app.DB().Update(
					(&teleblog.Post{}).TableName(),
					map[string]interface{}{
						"tg_group_message_id": c.Message().ID,
					},
					dbx.HashExp{"tg_post_id": c.Message().OriginalMessageID},
				).Execute()

				return err
			}

			newComment := &teleblog.Comment{
				ChatId:      chat.Id,
				Text:        c.Message().Text + c.Message().Caption,
				TgMessageId: c.Message().ID,
			}

			post := teleblog.Post{}

			// # Bind by thread id
			if c.Message().ThreadID > 0 {
				err := teleblog.PostQuery(app.Dao()).
					AndWhere(dbx.HashExp{"tg_group_message_id": c.Message().ThreadID}).
					Limit(1).
					One(&post)
				if err != nil && !strings.Contains(err.Error(), "no rows in result set") {
					return err
				}
			}

			if post.Id != "" {
				newComment.PostId = post.Id
			}

			newComment.Created.Scan(c.Message().Time())

			if c.Message().ReplyTo != nil {
				newComment.TgReplyToMessageId = c.Message().ReplyTo.ID
			}

			jsonMessageRaw, err := json.Marshal(c.Message())
			if err != nil {
				return err
			}

			err = newComment.TgMessageRaw.Scan(jsonMessageRaw)
			if err != nil {
				return err
			}

			err = app.Dao().Save(newComment)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("Unknown", c.Message().Text)
		}

		return nil
	})

	// # Edited messages in channels and groups
	b.Handle(telebot.OnEditedChannelPost, func(c telebot.Context) error {
		fmt.Println("OnEditedChannelPost")

		if skipContent(c) {
			return nil
		}

		chat := &teleblog.Chat{}

		err = teleblog.ChatQuery(app.Dao()).
			AndWhere(dbx.HashExp{"tg_chat_id": c.Chat().ID}).
			Limit(1).
			One(chat)
		if err != nil {
			return err
		}

		jsonMessageRaw, err := json.Marshal(c.Message())
		if err != nil {
			return err
		}

		var tgMessageRaw types.JsonMap

		err = tgMessageRaw.Scan(jsonMessageRaw)
		if err != nil {
			return err
		}

		post := teleblog.Post{}

		err = teleblog.PostQuery(app.Dao()).
			AndWhere(dbx.HashExp{"chat_id": chat.Id, "tg_post_id": c.Message().ID}).
			Limit(1).
			One(&post)
		if err != nil {
			return err
		}

		post.Text = c.Message().Text + c.Message().Caption
		post.TgMessageRaw = tgMessageRaw
		post.IsTgHistoryMessage = false

		err = app.Dao().Save(&post)
		if err != nil {
			return err
		}

		err = features.ExtractAndSavePostTags(app, post)
		if err != nil {
			return err
		}

		return err
	})

	b.Handle(telebot.OnEdited, func(c telebot.Context) error {
		fmt.Println("OnEdited", c.Message().Text+c.Message().Caption)
		fmt.Println("c.Sender().ID", c.Sender().ID)

		if skipContent(c) {
			return nil
		}

		chat := &teleblog.Chat{}

		err = teleblog.ChatQuery(app.Dao()).
			AndWhere(dbx.HashExp{"tg_chat_id": c.Chat().ID}).
			Limit(1).
			One(chat)
		if err != nil {
			return err
		}

		if c.Message().OriginalChat != nil && c.Message().OriginalChat.ID == chat.TgLinkedChatId {
			fmt.Println("FROM POST EDIT")
			return nil
		}

		jsonMessageRaw, err := json.Marshal(c.Message())
		if err != nil {
			return err
		}

		var tgMessageRaw types.JsonMap

		err = tgMessageRaw.Scan(jsonMessageRaw)
		if err != nil {
			return err
		}

		_, err = app.DB().Update(
			(&teleblog.Comment{}).TableName(),
			map[string]interface{}{
				"text":                  c.Message().Text,
				"tg_message_raw":        tgMessageRaw,
				"is_tg_history_message": false,
			},
			dbx.HashExp{"chat_id": chat.Id, "tg_comment_id": c.Message().ID},
		).Execute()

		return err
	})
}
