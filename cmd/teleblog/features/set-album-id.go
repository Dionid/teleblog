package features

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/pocketbase"
	"gopkg.in/telebot.v4"
)

func SetAlbumId(app *pocketbase.PocketBase) error {
	posts := []*teleblog.Post{}
	err := teleblog.
		PostQuery(app.Dao()).
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
				return fmt.Errorf("SetAlbumId: (post_id: %s) unmarshal history message error: %w", post.Id, err)
			}

			// if rawMessage.Photo == nil {
			// 	continue
			// }

			// This is a hack to set AlbumID to DateUnix
			post.AlbumID = rawMessage.DateUnix
		} else {
			rawMessage := telebot.Message{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				return fmt.Errorf("SetAlbumId: (post_id: %s) unmarshal realtime message error: %w", post.Id, err)
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
