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

		// remove
		collection.Schema.RemoveField("ezr0hqt7")

		// add
		new_logo_url := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "mtwrcdeu",
			"name": "logo_url",
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
		}`), new_logo_url); err != nil {
			return err
		}
		collection.Schema.AddField(new_logo_url)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g5axsrp0qjo62t9")
		if err != nil {
			return err
		}

		// add
		del_logo_url := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ezr0hqt7",
			"name": "logo_url",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_logo_url); err != nil {
			return err
		}
		collection.Schema.AddField(del_logo_url)

		// remove
		collection.Schema.RemoveField("mtwrcdeu")

		return dao.SaveCollection(collection)
	})
}
