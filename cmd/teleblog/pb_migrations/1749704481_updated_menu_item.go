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

		collection, err := dao.FindCollectionByNameOrId("ltd548lkltrvx4b")
		if err != nil {
			return err
		}

		// add
		new_position := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "yfyduktl",
			"name": "position",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_position); err != nil {
			return err
		}
		collection.Schema.AddField(new_position)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("ltd548lkltrvx4b")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("yfyduktl")

		return dao.SaveCollection(collection)
	})
}
