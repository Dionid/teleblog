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
		new_main_color := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "he4vnzlr",
			"name": "main_color",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_main_color); err != nil {
			return err
		}
		collection.Schema.AddField(new_main_color)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g5axsrp0qjo62t9")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("he4vnzlr")

		return dao.SaveCollection(collection)
	})
}
