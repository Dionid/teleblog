package httpapi

import (
	"encoding/json"

	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/telebot.v3"
)

type PostPageFilters struct {
	Page    int64  `query:"page"`
	PerPage int64  `query:"per_page"`
	Search  string `query:"search"`
	Tag     string `query:"tag"`
}

func baseQuery(
	app core.App,
	filters PostPageFilters,
	chatIds ...interface{},
) *dbx.SelectQuery {
	// Query
	baseQuery := teleblog.PostQuery(app.Dao()).
		LeftJoin(
			"comment",
			dbx.NewExp("comment.post_id = post.id"),
		).
		Where(
			dbx.In("post.chat_id", chatIds...),
		).
		// to avoid unsupported post types (video, photo, file, etc.)
		AndWhere(
			dbx.Or(
				dbx.NewExp(`post.text != ""`),
				dbx.NewExp(`json_array_length(post.photos) > 0`),
			),
		)

	// ## Filters

	if filters.Search != "" {
		baseQuery = baseQuery.AndWhere(
			dbx.Or(
				dbx.Like("post.text", filters.Search),
				dbx.Like("comment.text", filters.Search),
			),
		)
	}

	if filters.Tag != "" {
		baseQuery = baseQuery.
			LeftJoin(
				"post_tag",
				dbx.NewExp("post_tag.post_id = post.id"),
			).
			LeftJoin(
				"tag",
				dbx.NewExp("tag.id = post_tag.tag_id"),
			).
			AndWhere(
				dbx.HashExp{"tag.value": filters.Tag},
			)
	}

	return baseQuery
}

func IndexPageHandler(config Config, e *core.ServeEvent, app core.App) {
	e.Router.GET("", func(c echo.Context) error {
		chats := []teleblog.Chat{}

		err := teleblog.ChatQuery(app.Dao()).Where(
			dbx.HashExp{"user_id": config.UserId, "tg_type": "channel"},
		).All(&chats)
		if err != nil {
			return err
		}

		if len(chats) == 0 {
			// TODO: Change
			return c.JSON(200, []teleblog.Post{})
		}

		chatIds := []interface{}{}
		for _, chat := range chats {
			chatIds = append(chatIds, chat.Id)
		}

		// # Filters
		var filters PostPageFilters

		if err := c.Bind(&filters); err != nil {
			return err
		}

		// ## Total
		total := []struct {
			Total int64 `db:"total"`
		}{}

		err = baseQuery(
			app,
			filters,
			chatIds...,
		).Select(
			"count(post.id) as total",
		).
			GroupBy("post.album_id").
			All(&total)
		if err != nil {
			return err
		}

		// fmt.Println("TOTAL ", totalQuery.Build().SQL())

		// ## Posts
		posts := []*views.InpexPagePost{}
		contentQuery := baseQuery(
			app,
			filters,
			chatIds...,
		).Select(
			"post.*",
			"count(comment.id) as comments_count",
			"chat.tg_username as tg_chat_username",
			"json_group_array(json_object("+
				"'id', post.id,"+
				"'photos', post.photos"+
				")) as album_posts",
		).
			LeftJoin(
				"chat",
				dbx.NewExp("chat.id = post.chat_id"),
			).
			// TODO: think about it
			// GroupBy("post.id").
			OrderBy("post.created desc", "post.tg_post_id asc").
			GroupBy("post.album_id")

		// ## Pagination
		// ### Per page
		perPage := filters.PerPage

		if perPage == 0 {
			perPage = 10
		} else if perPage > 100 {
			perPage = 100
		}

		contentQuery = contentQuery.Limit(perPage)

		// ### Current page
		currentPage := filters.Page
		if currentPage == 0 {
			currentPage = 1
		}

		contentQuery = contentQuery.Offset((currentPage - 1) * perPage)

		err = contentQuery.
			All(&posts)
		if err != nil {
			return err
		}

		// TODO: count comments separately, because search string make it incorrect
		// ...

		postCollection, err := app.Dao().FindCollectionByNameOrId("post")
		if err != nil {
			return err
		}

		for _, post := range posts {
			markup := ""

			for i, photo := range post.Photos {
				post.Photos[i] = postCollection.Id + "/" + post.Id + "/" + photo
			}

			for _, innerPost := range post.AlbumPosts {
				if innerPost.Id == post.Id || innerPost.Photos == "" {
					continue
				}

				// # Text
				post.Text += innerPost.Text

				// # Photos
				photos := []string{}

				err = json.Unmarshal([]byte(innerPost.Photos), &photos)
				if err != nil {
					return err
				}

				for i, photo := range photos {
					photos[i] = postCollection.Id + "/" + innerPost.Id + "/" + photo
				}

				post.Photos = append(post.Photos, photos...)
			}

			// # Prase raw message
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

				markup = teleblog.FormHistoryTextWithMarkup(rawMessage.TextEntities)
			} else {
				rawMessage := telebot.Message{}

				err = json.Unmarshal(jb, &rawMessage)
				if err != nil {
					return err
				}

				markup, err = teleblog.FormWebhookTextMarkup(post.Text, rawMessage.Entities)
				if err != nil {
					return err
				}
			}

			post.TextWithMarkup = markup
		}

		// # Tags

		tags := []*teleblog.Tag{}

		err = teleblog.TagQuery(app.Dao()).
			LeftJoin(
				"post_tag",
				dbx.NewExp("post_tag.tag_id = tag.id"),
			).
			Where(
				dbx.In("post_tag.chat_id", chatIds...),
			).
			OrderBy("created desc").
			All(&tags)
		if err != nil {
			return err
		}

		pagination := views.PaginationData{
			Total:       int64(len(total)),
			PerPage:     perPage,
			CurrentPage: currentPage,
		}

		component := views.IndexPage(pagination, posts, tags)

		return component.Render(c.Request().Context(), c.Response().Writer)
	})
}
