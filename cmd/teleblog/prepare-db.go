package main

import (
	"fmt"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func prepareDB(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
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
			if strings.Contains(err.Error(), "no rows in result set") {
				return nil
			}
			return fmt.Errorf("Query existing tags error: %w", err)
		}

		if len(existingTags) > 0 {
			return nil
		}

		err = features.ExtractAndSaveAllTags(app)
		if err != nil {
			return fmt.Errorf("Extract and save all tags error: %w", err)
		}
		return nil
	})
}
