package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/net/html"
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
				dbx.NewExp(`json_array_length(post.media) > 0`),
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

func extractFirstURL(text string) string {
	urlRegex := regexp.MustCompile(`https?://[^\s<>"]+|www\.[^\s<>"]+`)
	matches := urlRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func fetchLinkPreview(url string) (*views.LinkPreview, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	preview := &views.LinkPreview{
		URL: url,
	}

	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					preview.Title = n.FirstChild.Data
				}
			case "meta":
				var property, content string
				for _, attr := range n.Attr {
					switch attr.Key {
					case "property", "name":
						property = attr.Val
					case "content":
						content = attr.Val
					}
				}
				switch property {
				case "og:title":
					if preview.Title == "" {
						preview.Title = content
					}
				case "og:description", "description":
					if preview.Description == "" {
						preview.Description = content
					}
				case "og:image":
					if preview.Image == "" {
						preview.Image = content
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	findMeta(doc)

	return preview, nil
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
				"'media', post.media"+
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

			for i, media := range post.Media {
				post.Media[i] = postCollection.Id + "/" + post.Id + "/" + media
			}

			for _, innerPost := range post.AlbumPosts {
				if innerPost.Id == post.Id || innerPost.Media == "" {
					continue
				}

				// # Text
				post.Text += innerPost.Text

				// # Photos
				medias := []string{}

				err = json.Unmarshal([]byte(innerPost.Media), &medias)
				if err != nil {
					return err
				}

				for i, media := range medias {
					medias[i] = postCollection.Id + "/" + innerPost.Id + "/" + media
				}

				post.Media = append(post.Media, medias...)
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

			// Extract and fetch link preview
			if url := extractFirstURL(post.Text); url != "" {
				if preview, err := fetchLinkPreview(url); err == nil {
					post.LinkPreview = preview
				}
			}
		}

		// # Tags

		tags := []*teleblog.Tag{}

		err = teleblog.TagQuery(app.Dao()).
			Select("tag.value").
			LeftJoin(
				"post_tag",
				dbx.NewExp("post_tag.tag_id = tag.id"),
			).
			Where(
				dbx.In("post_tag.chat_id", chatIds...),
			).
			OrderBy("tag.created desc").
			GroupBy("tag.value").
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
