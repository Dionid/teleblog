package botapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/slug"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/Dionid/teleblog/libs/templu"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
	"gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

const ADD_CHANNEL_COMMAND_NAME = "addchannel"
const VERIFY_TOKEN_COMMAND_NAME = "verifytoken"

func skipContent(_ telebot.Context) bool {
	// # We can't skip content, because we need all posts for links
	return false
}

// downloadPhoto downloads a photo from a message to the specified directory
func downloadPhoto(b *telebot.Bot, fileId string, uniqueID string, outputDir string, fileExt string) (string, error) {
	// Get file info from Telegram
	file, err := b.FileByID(fileId)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Generate unique filename
	filename := filepath.Join(outputDir, fmt.Sprintf("%s.%s", uniqueID, fileExt))

	// Download the file
	err = b.Download(&file, filename)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	return filename, nil
}

func InitBotCommands(b *telebot.Bot, app *pocketbase.PocketBase) error {
	err := b.SetCommands([]telebot.Command{
		{Text: "start", Description: "start the bot"},
		{Text: VERIFY_TOKEN_COMMAND_NAME, Description: "send token to bind bot to your telebot account (e.g. /verifytoken YOUR_TOKEN)"},
		{Text: ADD_CHANNEL_COMMAND_NAME, Description: "send channel to create blog from it (e.g. /addchannel @YOUR_CHANNEL_NAME)"},
	})
	if err != nil {
		return err
	}

	b.Handle("/start", func(c telebot.Context) error {
		return c.Reply("Hello! This is teleblog bot. Add it to your channel and get posts in your blog.")
	})

	b.Use(middleware.Recover(func(err error, ctx telebot.Context) {
		app.Logger().Error("Error in bot: ", err)
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

		rawMessage := c.Message()

		text := rawMessage.Text + rawMessage.Caption

		newPost := &teleblog.Post{
			ChatId:         chat.Id,
			IsTgMessage:    true,
			Text:           text,
			TgMessageId:    rawMessage.ID,
			AlbumID:        rawMessage.AlbumID,
			Title:          templu.RemoveNewLines(fmt.Sprintf("%.60s", text)),
			SeoDescription: templu.RemoveNewLines(fmt.Sprintf("%.160s", text)),
			Slug:           slug.GenerateSlug(text, time.Now()),
		}

		if newPost.AlbumID == "" {
			newPost.AlbumID = strconv.Itoa(int(rawMessage.Unixtime))
		}

		newPost.Created.Scan(rawMessage.Time())

		jsonMessageRaw, err := json.Marshal(rawMessage)
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

		postCollection, err := app.Dao().FindCollectionByNameOrId("post")
		if err != nil {
			return err
		}

		fsys, err := app.NewFilesystem()
		if err != nil {
			return err
		}
		defer fsys.Close()

		// Handle photo if present
		if photo := c.Message().Photo; photo != nil {
			outputDir := "temp-tg-webhook-uploads-" + newPost.Id
			// Create output directory if it doesn't exist
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			defer os.RemoveAll(outputDir)

			filename, err := downloadPhoto(b, photo.FileID, photo.UniqueID, outputDir, "jpg")
			if err != nil {
				return fmt.Errorf("failed to download photo: %w", err)
			}

			file, err := filesystem.NewFileFromPath(filename)
			if err != nil {
				return err
			}

			fileName := postCollection.Id + "/" + newPost.Id + "/" + file.Name

			err = fsys.UploadFile(file, fileName)
			if err != nil {
				return err
			}

			newPost.Media = append(newPost.Media, file.Name)
		}

		if video := c.Message().Video; video != nil {
			outputDir := "temp-tg-webhook-uploads-" + newPost.Id
			// Create output directory if it doesn't exist
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			defer os.RemoveAll(outputDir)

			filename, err := downloadPhoto(b, video.FileID, video.UniqueID, outputDir, "mp4")
			if err != nil {
				return fmt.Errorf("failed to download photo: %w", err)
			}

			file, err := filesystem.NewFileFromPath(filename)
			if err != nil {
				return err
			}

			fileName := postCollection.Id + "/" + newPost.Id + "/" + file.Name

			err = fsys.UploadFile(file, fileName)
			if err != nil {
				return err
			}

			newPost.Media = append(newPost.Media, file.Name)
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
			if c.Message().ThreadID != 0 {
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

		rawMessage := c.Message()

		jsonMessageRaw, err := json.Marshal(rawMessage)
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
			AndWhere(dbx.HashExp{"chat_id": chat.Id, "tg_post_id": rawMessage.ID}).
			Limit(1).
			One(&post)
		if err != nil {
			return err
		}

		text := rawMessage.Text + rawMessage.Caption

		post.Text = text
		post.TgMessageRaw = tgMessageRaw
		post.IsTgHistoryMessage = false
		post.AlbumID = rawMessage.AlbumID

		if post.AlbumID == "" {
			post.AlbumID = strconv.Itoa(int(rawMessage.Unixtime))
		}

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
		rawMessage := c.Message()

		fmt.Println("OnEdited", rawMessage.Text+rawMessage.Caption)
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

		if rawMessage.OriginalChat != nil && rawMessage.OriginalChat.ID == chat.TgLinkedChatId {
			fmt.Println("FROM POST EDIT")
			return nil
		}

		jsonMessageRaw, err := json.Marshal(rawMessage)
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
				"text":                  rawMessage.Text,
				"tg_message_raw":        tgMessageRaw,
				"is_tg_history_message": false,
			},
			dbx.HashExp{"chat_id": chat.Id, "tg_comment_id": rawMessage.ID},
		).Execute()

		return err
	})

	return nil
}
