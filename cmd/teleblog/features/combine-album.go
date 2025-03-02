package features

import (
	"encoding/json"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/pocketbase"
	"gopkg.in/telebot.v3"
)

func MergePostsByAlbum(app *pocketbase.PocketBase) error {
	posts := []*teleblog.Post{}
	err := teleblog.
		PostQuery(app.Dao()).
		OrderBy("tg_post_id").
		All(&posts)

	byAlbumId := map[string][]*teleblog.Post{}

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
		} else {
			rawMessage := telebot.Message{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				return err
			}

			if rawMessage.AlbumID != "" {
				byAlbumId[rawMessage.AlbumID] = append(byAlbumId[rawMessage.AlbumID], post)
			}
		}
	}

	originals := []*teleblog.Post{}
	toDelete := []*teleblog.Post{}

	for _, v := range byAlbumId {
		if len(v) < 2 {
			continue
		}

		original := v[0]
		originals = append(originals, original)

		for i, post := range v {
			if i == 0 {
				continue
			}

			original.Photos = append(original.Photos, post.Photos...)
			toDelete = append(toDelete, post)
		}
	}

	// for _, p := range originals {
	// 	app.Dao().Save(p)
	// }

	// for _, p := range toDelete {
	// 	app.Dao().Delete(p)
	// }

	if err != nil {
		return err
	}

	return nil
}
