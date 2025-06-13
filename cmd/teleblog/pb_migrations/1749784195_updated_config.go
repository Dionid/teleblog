package pb_migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g5axsrp0qjo62t9")
		if err != nil {
			return err
		}

		// add
		new_favicon := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "rjemriwo",
			"name": "favicon",
			"type": "file",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"mimeTypes": [],
				"thumbs": [],
				"maxSelect": 1,
				"maxSize": 5242880,
				"protected": false
			}
		}`), new_favicon); err != nil {
			return err
		}
		collection.Schema.AddField(new_favicon)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g5axsrp0qjo62t9")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("rjemriwo")

		return dao.SaveCollection(collection)
	})
}
