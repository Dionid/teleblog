package features

import (
	"strings"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

func ExtractAndSavePostTags(app *pocketbase.PocketBase, post teleblog.Post) error {
	tags, err := teleblog.ExtractTagsFromPost(post)
	if err != nil {
		if strings.Contains(err.Error(), "unmarshal message") {
			return nil
		}
		return err
	}

	for _, tagValue := range tags {
		tag := teleblog.Tag{
			Value: tagValue,
		}

		err := app.Dao().Save(&tag)
		if err != nil {
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return err
			}
			err = nil

			err = teleblog.TagQuery(app.Dao()).
				Where(dbx.HashExp{"value": tag.Value}).
				One(&tag)
			if err != nil {
				return err
			}
		}

		postTag := teleblog.PostTag{
			TagId:  tag.Id,
			PostId: post.Id,
			ChatId: post.ChatId,
		}

		err = app.Dao().Save(&postTag)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				continue
			}
			return err
		}
	}

	return nil
}

func ExtractAndSaveAllTags(app *pocketbase.PocketBase) error {
	var posts []teleblog.Post

	err := teleblog.PostQuery(app.Dao()).
		OrderBy("created ASC").
		Where(dbx.HashExp{"unparsable": false}).
		All(&posts)
	if err != nil {
		return err
	}

	for _, post := range posts {
		err := ExtractAndSavePostTags(app, post)
		if err != nil {
			return err
		}
	}

	return nil
}
