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
				dbx.NewExp(`post.text != ""`),
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

		// ## Total
		total := []struct {
			Total int64 `db:"total"`
		}{}

		err = baseQuery.Select(
			"count(post.id) as total",
		).
			GroupBy("post.id").
			All(&total)
		if err != nil {
			return err
		}

		// ## Posts
		posts := []*views.InpexPagePost{}
		contentQuery := baseQuery.Select(
			"post.*",
			"count(comment.id) as comments_count",
			"chat.tg_username as tg_chat_username",
		).
			LeftJoin(
				"chat",
				dbx.NewExp("chat.id = post.chat_id"),
			).
			GroupBy("post.id").
			OrderBy("post.created desc")

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

			for i, photo := range post.Photos {
				post.Photos[i] = postCollection.Id + "/" + post.Id + "/" + photo
			}
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
			GroupBy("tag.id").
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
