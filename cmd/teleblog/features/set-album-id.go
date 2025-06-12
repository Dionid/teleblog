package features

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"gopkg.in/telebot.v4"
)

func SetAlbumId(app *pocketbase.PocketBase) error {
	posts := []*teleblog.Post{}
	err := teleblog.
		PostQuery(app.Dao()).
		Where(dbx.HashExp{"unparsable": false}).
		All(&posts)

	for _, post := range posts {
		jb, err := post.TgMessageRaw.MarshalJSON()
		if err != nil {
			return err
		}

		if post.IsTgHistoryMessage {
			rawMessage := teleblog.HistoryMessage{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				app.Logger().Error("SetAlbumId: unmarshal history message error", "error", err, "post_id", post.Id)
				_, err := app.DB().Update(
					"post",
					dbx.Params{"unparsable": true},
					dbx.HashExp{"id": post.Id},
				).Execute()
				if err != nil {
					return fmt.Errorf("IndexPageHandler: update post error: %w", err)
				}
				continue
			}

			// This is a hack to set AlbumID to DateUnix
			post.AlbumID = rawMessage.DateUnix
		} else {
			rawMessage := telebot.Message{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				app.Logger().Error("SetAlbumId: unmarshal realtime message error", "error", err, "post_id", post.Id)
				_, err := app.DB().Update(
					"post",
					dbx.Params{"unparsable": true},
					dbx.HashExp{"id": post.Id},
				).Execute()
				if err != nil {
					return fmt.Errorf("IndexPageHandler: update post error: %w", err)
				}
				continue
			}

			if rawMessage.AlbumID != "" {
				post.AlbumID = rawMessage.AlbumID
			} else {
				post.AlbumID = strconv.Itoa(int(rawMessage.Unixtime))
			}
		}

		err = app.Dao().Save(post)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	return nil
}
