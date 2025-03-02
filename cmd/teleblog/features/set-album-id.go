package features

import (
	"encoding/json"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/pocketbase"
	"gopkg.in/telebot.v3"
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
				return err
			}

			if rawMessage.Photo == nil {
				continue
			}

			// This is a hack to set AlbumID to DateUnix
			post.AlbumID = rawMessage.DateUnix
		} else {
			rawMessage := telebot.Message{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				return err
			}

			post.AlbumID = rawMessage.AlbumID
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
