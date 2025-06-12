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
		new_yandex_metrika_counter := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5qflifel",
			"name": "yandex_metrika_counter",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_yandex_metrika_counter); err != nil {
			return err
		}
		collection.Schema.AddField(new_yandex_metrika_counter)

		// add
		new_google_analytics_counter := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tojzn93r",
			"name": "google_analytics_counter",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_google_analytics_counter); err != nil {
			return err
		}
		collection.Schema.AddField(new_google_analytics_counter)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g5axsrp0qjo62t9")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("5qflifel")

		// remove
		collection.Schema.RemoveField("tojzn93r")

		return dao.SaveCollection(collection)
	})
}
