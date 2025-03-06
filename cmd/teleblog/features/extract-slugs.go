package features

import (
	"github.com/Dionid/teleblog/libs/slug"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/pocketbase"
)

func ExtractSlugs(app *pocketbase.PocketBase) error {
	// # Extract slugs
	var posts []teleblog.Post

	err := teleblog.PostQuery(app.Dao()).
		All(&posts)
	if err != nil {
		return err
	}

	for _, post := range posts {
		if post.Slug != "" {
			continue
		}

		if post.Title != "" {
			post.Slug = slug.GenerateSlug(post.Title, post.Created.Time())
		} else if post.Text != "" {
			post.Slug = slug.GenerateSlug(post.Text, post.Created.Time())
		} else {
			post.Slug = slug.GenerateSlug(post.Id, post.Created.Time())
		}

		err := app.Dao().Save(&post)
		if err != nil {
			return err
		}
	}

	return nil
}
