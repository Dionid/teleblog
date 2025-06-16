package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views"
	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views/partials"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/net/html"
	"gopkg.in/telebot.v4"
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
		).
		AndWhere(
			dbx.NewExp("post.unparsable = false"),
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
		// # Config
		siteConfig := teleblog.Config{}

		configCollection, err := teleblog.Configcollection(app.Dao())
		if err != nil {
			return fmt.Errorf("IndexPageHandler: get config collection error: %w", err)
		}

		err = teleblog.ConfigQuery(app.Dao()).One(&siteConfig)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				return c.JSON(404, map[string]string{
					"error": "Configuration not found",
				})
			}

			return err
		}

		if siteConfig.Id == "" {
			return c.JSON(404, map[string]string{
				"error": "Configuration not found",
			})
		}

		// # Get menu
		menu := []teleblog.MenuItem{}

		err = teleblog.MenuItemQuery(app.Dao()).OrderBy("position").All(&menu)
		if err != nil {
			return err
		}

		// # Get chats
		chats := []teleblog.Chat{}

		err = teleblog.ChatQuery(app.Dao()).Where(
			dbx.HashExp{"tg_type": "channel"},
		).All(&chats)
		if err != nil {
			return err
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
			"post.id",
			"post.album_id",
			"chat.id as chat_id",
			"post.tg_post_id",
			"post.tg_group_message_id",
			"post.created",
			"chat.tg_username as tg_chat_username",
		).
			LeftJoin(
				"chat",
				dbx.NewExp("chat.id = post.chat_id"),
			).
			GroupBy("post.album_id").
			OrderBy("post.created desc", "post.tg_post_id asc")

		// ## Pagination
		// ### Per page
		perPage := filters.PerPage

		if perPage == 0 {
			perPage = 10
		} else if perPage > 100 {
			perPage = 100
		}

		contentQuery = contentQuery.Limit(perPage)

		// ## Current page
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

		postCollection, err := app.Dao().FindCollectionByNameOrId("post")
		if err != nil {
			return err
		}

		for _, post := range posts {
			type InnerPost struct {
				teleblog.Post
				CommentsCount int `db:"comments_count"`
			}

			innerPosts := []InnerPost{}

			// THIS MUST CONTAIN ORIGINAL POST
			err := teleblog.PostQuery(app.Dao()).
				Select(
					"post.*",
					"count(comment.id) as comments_count",
				).
				LeftJoin(
					"comment",
					dbx.NewExp("comment.post_id = post.id"),
				).
				Where(
					dbx.HashExp{"post.album_id": post.AlbumID},
				).
				GroupBy("post.id").
				All(&innerPosts)
			if err != nil {
				return fmt.Errorf("IndexPageHandler: get inner posts error: %w", err)
			}

			for i, media := range post.Media {
				post.Media[i] = postCollection.Id + "/" + post.Id + "/" + media
			}

			for _, innerPost := range innerPosts {
				if innerPost.Text != "" {
					post.Slug = innerPost.Slug
				}

				// # Text
				post.Text += innerPost.Text + "\n\n"

				if innerPost.IsTgHistoryMessage {
					post.IsTgHistoryMessage = innerPost.IsTgHistoryMessage
				}

				// # Photos
				medias := []string{}

				for _, media := range innerPost.Media {
					medias = append(medias, postCollection.Id+"/"+innerPost.Id+"/"+media)
				}

				post.Media = append(post.Media, medias...)

				// # Markup
				markup := ""

				jb, err := innerPost.TgMessageRaw.MarshalJSON()
				if err != nil {
					return err
				}

				if innerPost.IsTgHistoryMessage {
					rawMessage := teleblog.HistoryMessage{}

					err = json.Unmarshal(jb, &rawMessage)
					if err != nil {
						app.Logger().Error("IndexPageHandler: unmarshal history message error", "error", err, "post_id", post.Id)
						_, err := app.DB().Update(
							"post",
							dbx.Params{"unparsable": true},
							dbx.HashExp{"id": innerPost.Id},
						).Execute()
						if err != nil {
							return fmt.Errorf("IndexPageHandler: update post error: %w", err)
						}
						continue
					}

					if len(rawMessage.Text.Items) > 0 {
						markup = teleblog.FormHistoryRawTextWithMarkup(rawMessage.Text)
					} else {
						markup = teleblog.HistoryTextEntitiesWithToTextWithMarkup(rawMessage.TextEntities)
					}
				} else {
					rawMessage := telebot.Message{}

					err = json.Unmarshal(jb, &rawMessage)
					if err != nil {
						app.Logger().Error("IndexPageHandler: unmarshal history message error", "error", err, "post_id", post.Id)
						_, err := app.DB().Update(
							"post",
							dbx.Params{"unparsable": true},
							dbx.HashExp{"id": innerPost.Id},
						).Execute()
						if err != nil {
							return fmt.Errorf("IndexPageHandler: update post error: %w", err)
						}
						continue // Skip if unmarshal error, it may be a non-history message
					}

					if len(rawMessage.Entities) > 0 {
						markup, err = teleblog.FormWebhookTextMarkup(rawMessage.Text, rawMessage.Entities)
						if err != nil {
							return err
						}
					} else if len(rawMessage.CaptionEntities) > 0 {
						markup, err = teleblog.FormWebhookTextMarkup(rawMessage.Caption, rawMessage.CaptionEntities)
						if err != nil {
							return err
						}
					}
				}

				post.TextWithMarkup += markup

				// # Comments count
				post.CommentsCount += innerPost.CommentsCount
			}

			post.Text = strings.ReplaceAll(post.Text, "\n", "<br>")

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

		// # Render component
		// ## Header
		header := partials.HeaderData{
			LogoUrl: teleblog.ImagePath(
				configCollection,
				&siteConfig.BaseModel,
				siteConfig.LogoUrl,
			),
			LogoAlt:   siteConfig.LogoAlt,
			MenuItems: []partials.HeaderMenuItem{},
		}

		for _, item := range menu {
			header.MenuItems = append(header.MenuItems, partials.HeaderMenuItem{
				Name: item.Name,
				Url:  item.Url,
			})
		}

		component := views.IndexPage(
			views.BaseLayoutData{
				Seo: views.SeoMetadata{
					Title:       siteConfig.SeoTitle,
					Description: siteConfig.SeoDescription,
					Image: teleblog.ImagePath(
						configCollection,
						&siteConfig.BaseModel,
						siteConfig.SeoImage,
					),
					Url:  siteConfig.SeoUrl,
					Type: "website",
				},
				YandexMetrikaCounter:   siteConfig.YandexMetrikaCounter,
				GoogleAnalyticsCounter: siteConfig.GoogleAnalyticsCounter,
				PrimaryColor:           siteConfig.PrimaryColor,
				BgImage: teleblog.ImagePath(
					configCollection,
					&siteConfig.BaseModel,
					siteConfig.BgImage,
				),
				FavIcon: teleblog.ImagePath(
					configCollection,
					&siteConfig.BaseModel,
					siteConfig.Favicon,
				),
				CustomCss: siteConfig.CustomCss,
			},
			views.IndexPageInfo{
				Description: siteConfig.Description,
				SelectedTag: filters.Tag,
				TextSearch:  filters.Search,
				Header:      header,
				Footer: partials.FooterData{
					Text: siteConfig.Footer,
				},
			},
			pagination,
			posts,
			tags,
		)

		return component.Render(c.Request().Context(), c.Response().Writer)
	})
}
