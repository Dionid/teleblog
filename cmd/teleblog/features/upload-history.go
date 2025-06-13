package features

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Dionid/teleblog/libs/slug"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/Dionid/teleblog/libs/templu"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func ParseChannelHistory(app core.App, historyZip teleblog.HistoryExport, history teleblog.History, chat *teleblog.Chat) error {
	fsys, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fsys.Close()

	postCollection, err := app.Dao().FindCollectionByNameOrId("post")
	if err != nil {
		return err
	}

	for _, message := range history.Messages {
		if message.Type != "message" {
			continue
		}

		// # Skip if exists
		total := struct {
			Total int64 `db:"total"`
		}{}

		err := teleblog.PostQuery(app.Dao()).
			Where(
				dbx.HashExp{"tg_post_id": message.Id, "chat_id": chat.Id},
			).
			Select(
				"count(*) as total",
			).
			One(&total)
		if err != nil {
			return err
		}

		if total.Total > 0 {
			continue
		}

		// # Extract text
		text := ""

		for _, entity := range message.TextEntities {
			text += entity.Text
		}

		// # Create new
		post := teleblog.Post{
			ChatId:             chat.Id,
			IsTgMessage:        true,
			IsTgHistoryMessage: true,
			Text:               text,
			TgMessageId:        message.Id,
			Title:              templu.RemoveNewLines(fmt.Sprintf("%.60s", text)),
			SeoDescription:     templu.RemoveNewLines(fmt.Sprintf("%.160s", text)),
			Slug:               slug.GenerateSlug(text, time.Now()),
		}

		// # post.Created
		if message.DateUnix != "" {
			i, err := strconv.ParseInt(message.DateUnix, 10, 64)
			if err != nil {
				return err
			}
			tm := time.Unix(i, 0)
			post.Created.Scan(tm)
		}

		// # post.TgMessageRaw
		jsonMessageRaw, err := json.Marshal(message)
		if err != nil {
			return err
		}

		err = post.TgMessageRaw.Scan(jsonMessageRaw)
		if err != nil {
			return err
		}

		err = app.Dao().Save(&post)
		if err != nil {
			return err
		}

		// # Hack for albums
		post.AlbumID = message.DateUnix

		if message.Photo != nil {
			for _, photoPath := range historyZip.Photos {
				if strings.Contains(photoPath, *message.Photo) {
					file, err := filesystem.NewFileFromPath(photoPath)
					if err != nil {
						return err
					}

					fileName := postCollection.Id + "/" + post.Id + "/" + file.Name

					err = fsys.UploadFile(file, fileName)
					if err != nil {
						return err
					}

					post.Media = append(post.Media, file.Name)
					break
				}
			}
		}

		if message.MediaType == "video_file" && message.File != nil {
			for _, videoPath := range historyZip.VideoFiles {
				if strings.Contains(videoPath, *message.File) {
					file, err := filesystem.NewFileFromPath(videoPath)
					if err != nil {
						return err
					}

					fileName := postCollection.Id + "/" + post.Id + "/" + file.Name

					fmt.Println("fileName: ", fileName)

					err = fsys.UploadFile(file, fileName)
					if err != nil {
						return err
					}

					post.Media = append(post.Media, file.Name)
					break
				}
			}
		}

		err = app.Dao().Save(&post)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseGroupHistory(app core.App, history teleblog.History, chat *teleblog.Chat) error {
	var preparedComments []teleblog.Comment

	for _, message := range history.Messages {
		if message.Type != "message" {
			continue
		}

		if message.Text == "" {
			continue
		}

		// # Skip if exists
		total := struct {
			Total int64 `db:"total"`
		}{}

		err := teleblog.CommentQuery(app.Dao()).
			Where(
				dbx.HashExp{"tg_comment_id": message.Id, "chat_id": chat.Id},
			).
			Select(
				"count(*) as total",
			).
			One(&total)
		if err != nil {
			return err
		}

		if total.Total > 0 {
			continue
		}

		// # Extract created time
		messageCreatedAt := time.Time{}

		if message.DateUnix != "" {
			i, err := strconv.ParseInt(message.DateUnix, 10, 64)
			if err != nil {
				return err
			}
			messageCreatedAt = time.Unix(i, 0)
		}

		// # If this is forward, than can be source post
		if message.ForwardedFrom != nil {
			forwardFromTgId, err := strconv.ParseInt(
				fmt.Sprintf(
					"-100%s",
					strings.ReplaceAll(message.FromId, "channel", ""),
				),
				10,
				64,
			)
			if err != nil {
				return err
			}

			// # If forward from linked chat, than it is source post
			if chat.TgLinkedChatId == forwardFromTgId {
				// # Find post and update its tg_message_id
				sourcePost := teleblog.Post{}

				err := teleblog.PostQuery(app.Dao()).
					Where(
						dbx.HashExp{"chat_id": chat.LinkedChatId},
					).
					AndWhere(
						dbx.NewExp("created <= {:t}", dbx.Params{"t": messageCreatedAt.UTC().Format("2006-01-02 15:04:05")}),
					).
					OrderBy("created DESC").
					Limit(1).
					One(&sourcePost)
				if err != nil {
					return err
				}

				if sourcePost.TgGroupMessageId != 0 {
					continue
				}

				_, err = app.DB().Update(
					(&teleblog.Post{}).TableName(),
					map[string]interface{}{
						"tg_group_message_id": message.Id,
					},
					dbx.HashExp{"id": sourcePost.Id},
				).Execute()
				if err != nil {
					return err
				}

				continue
			}
		}

		// // # Skip if it is not reply to something
		// if message.ReplyToMessageId == 0 {
		// 	continue
		// }

		// # Get reply post only if it reply
		post := teleblog.Post{}

		if message.ReplyToMessageId != 0 {
			err := teleblog.PostQuery(app.Dao()).
				Where(
					dbx.HashExp{"tg_group_message_id": message.ReplyToMessageId, "chat_id": chat.LinkedChatId},
				).
				Limit(1).
				One(&post)
			if err != nil {
				if !strings.Contains(err.Error(), "no rows in result set") {
					return err
				}

				var parentComment *teleblog.Comment

				// # Find parent comment in prepared
				for _, comment := range preparedComments {
					if comment.TgMessageId == message.ReplyToMessageId {
						parentComment = &comment
					}
				}

				// # If none, than find it in DB
				if parentComment == nil {
					err := teleblog.CommentQuery(app.Dao()).
						Where(
							dbx.HashExp{"tg_comment_id": message.ReplyToMessageId, "chat_id": chat.Id},
						).
						Limit(1).
						One(parentComment)
					if err != nil {
						if strings.Contains(err.Error(), "no rows in result set") {
							continue
						}
						return err
					}
				}

				if parentComment != nil && parentComment.PostId != "" {
					err := teleblog.PostQuery(app.Dao()).
						Where(
							dbx.HashExp{"id": parentComment.PostId},
						).
						One(&post)
					if err != nil {
						return err
					}
				}
			}
		}

		// # Extract text
		text := ""

		for _, entity := range message.TextEntities {
			text += entity.Text
		}

		// # Create new
		comment := teleblog.Comment{
			ChatId:             chat.Id,
			Text:               text,
			TgMessageId:        message.Id,
			TgReplyToMessageId: message.ReplyToMessageId,
			IsTgHistoryMessage: true,
		}

		if post.Id != "" {
			comment.PostId = post.Id
		}

		comment.Created.Scan(messageCreatedAt)

		// # post.TgMessageRaw
		jsonMessageRaw, err := json.Marshal(message)
		if err != nil {
			return err
		}

		err = comment.TgMessageRaw.Scan(jsonMessageRaw)
		if err != nil {
			return err
		}

		preparedComments = append(preparedComments, comment)
	}

	// # Save
	if len(preparedComments) == 0 {
		return nil
	}

	for _, comment := range preparedComments {
		err := app.Dao().Save(&comment)
		if err != nil {
			return err
		}
	}

	return nil
}

func UploadHistory(app *pocketbase.PocketBase, historyExportPath string) error {
	// Parse zip structure
	structure, err := teleblog.FolderToHistoryExport(historyExportPath)
	if err != nil {
		return fmt.Errorf("failed to parse zip structure: %v", err)
	}

	resultFile, err := os.Open(structure.ResultJson)
	if err != nil {
		return err
	}
	defer resultFile.Close()

	// Parse JSON content from result.json
	var history teleblog.History
	if err := json.NewDecoder(resultFile).Decode(&history); err != nil {
		return fmt.Errorf("invalid JSON format in result.json: %v", err)
	}

	chatId, err := history.GetChatTgId()
	if err != nil {
		return err
	}

	var chat teleblog.Chat

	err = teleblog.ChatQuery(app.Dao()).Where(
		dbx.HashExp{"tg_chat_id": chatId},
	).Limit(1).One(&chat)
	if err != nil {
		return err
	}

	if chat.TgType == "channel" {
		err := ParseChannelHistory(app, *structure, history, &chat)
		if err != nil {
			return err
		}

		err = ExtractAndSaveAllTags(app)
		if err != nil {
			return err
		}

		return nil
	} else if chat.TgType == "supergroup" || chat.TgType == "group" {
		return ParseGroupHistory(app, history, &chat)
	}

	return nil
}
