package httpapi

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views"
	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views/partials"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/Dionid/teleblog/libs/templu"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/telebot.v4"
)

func PostPageHandler(e *core.ServeEvent, app core.App) {
	e.Router.GET("/post/:id", func(c echo.Context) error {
		// # Site config collection
		siteConfigCollection, err := teleblog.Configcollection(app.Dao())
		if err != nil {
			return fmt.Errorf("IndexPageHandler: get config collection error: %w", err)
		}

		// # Config
		siteConfig := teleblog.Config{}

		err = teleblog.ConfigQuery(app.Dao()).One(&siteConfig)
		if err != nil {
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

		// # Get post by ID or slug
		postIdOrSlug := c.PathParam("id")

		post := views.PostPagePost{}
		err = teleblog.PostQuery(app.Dao()).Where(
			dbx.Or(
				dbx.HashExp{"id": postIdOrSlug},
				dbx.HashExp{"slug": postIdOrSlug},
			),
		).AndWhere(
			dbx.NewExp("unparsable = false"),
		).Limit(1).One(&post)
		if err != nil {
			return err
		}

		// # Correct post media URLs
		postCollection, err := app.Dao().FindCollectionByNameOrId("post")
		if err != nil {
			return err
		}

		for i, media := range post.Media {
			post.Media[i] = "/api/files/" + postCollection.Id + "/" + post.Id + "/" + media
		}

		// # Get album posts
		albumPosts := []*views.PostPagePost{}
		err = teleblog.PostQuery(app.Dao()).Where(
			dbx.HashExp{"album_id": post.AlbumID},
		).AndWhere(
			dbx.Not(
				dbx.HashExp{"id": post.Id},
			),
		).All(&albumPosts)
		if err != nil {
			return err
		}

		// # Correct media URLs for album posts
		for _, albumPost := range albumPosts {
			for i, media := range albumPost.Media {
				albumPost.Media[i] = "/api/files/" + postCollection.Id + "/" + albumPost.Id + "/" + media
			}

			post.Media = append(post.Media, albumPost.Media...)
		}

		// # Remarshal JSON to correct type
		jb, err := post.Post.TgMessageRaw.MarshalJSON()
		if err != nil {
			return err
		}

		// # Text with markup
		if post.IsTgHistoryMessage {
			rawMessage := teleblog.HistoryMessage{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				return err
			}

			if len(rawMessage.Text.Items) > 0 {
				post.TextWithMarkup = teleblog.FormHistoryRawTextWithMarkup(rawMessage.Text)
			} else if len(rawMessage.TextEntities) > 0 {
				post.TextWithMarkup = teleblog.HistoryTextEntitiesWithToTextWithMarkup(rawMessage.TextEntities)
			} else {
				post.TextWithMarkup = strings.ReplaceAll(
					html.EscapeString(rawMessage.Title),
					"\n",
					"<br>",
				)
			}
		} else {
			rawMessage := telebot.Message{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				return err
			}

			if len(rawMessage.Entities) > 0 {
				post.TextWithMarkup, err = teleblog.FormWebhookTextMarkup(rawMessage.Text, rawMessage.Entities)
				if err != nil {
					return err
				}
			} else if len(rawMessage.CaptionEntities) > 0 {
				post.TextWithMarkup, err = teleblog.FormWebhookTextMarkup(rawMessage.Caption, rawMessage.CaptionEntities)
				if err != nil {
					return err
				}
			} else {
				post.TextWithMarkup = strings.ReplaceAll(
					html.EscapeString(rawMessage.Text),
					"\n",
					"<br>",
				)
			}
		}

		// # Get comments from group chat
		chat := teleblog.Chat{}

		err = teleblog.ChatQuery(app.Dao()).Where(
			dbx.HashExp{"id": post.ChatId},
		).Limit(1).One(&chat)
		if err != nil {
			return err
		}

		comments := []*views.PostPageComment{}

		err = teleblog.CommentQuery(app.Dao()).Where(
			dbx.HashExp{"post_id": post.Id},
		).All(&comments)
		if err != nil {
			return err
		}

		// # Prepare comments
		for _, comment := range comments {
			jb, err := comment.TgMessageRaw.MarshalJSON()
			if err != nil {
				return fmt.Errorf("PostPageHandler: marshal comment error: %w", err)
			}

			if comment.IsTgHistoryMessage {
				rawMessage := teleblog.HistoryMessage{}

				err = json.Unmarshal(jb, &rawMessage)
				if err != nil {
					return fmt.Errorf("PostPageHandler: unmarshal history message error: %w", err)
				}

				comment.AuthorTitle = rawMessage.From

				if len(rawMessage.Text.Items) > 0 {
					comment.TextWithMarkup = teleblog.FormHistoryRawTextWithMarkup(rawMessage.Text)
				} else {
					comment.TextWithMarkup = teleblog.HistoryTextEntitiesWithToTextWithMarkup(rawMessage.TextEntities)
				}
			} else {
				rawMessage := telebot.Message{}

				err = json.Unmarshal(jb, &rawMessage)
				if err != nil {
					return err
				}

				if rawMessage.Sender.IsBot && rawMessage.SenderChat != nil {
					comment.AuthorTitle = rawMessage.SenderChat.Title
					comment.AuthorUsername = &rawMessage.SenderChat.Username
				} else {
					comment.AuthorTitle = rawMessage.Sender.FirstName + " " + rawMessage.Sender.LastName
					comment.AuthorUsername = &rawMessage.Sender.Username
				}

				if len(rawMessage.Entities) > 0 {
					comment.TextWithMarkup, err = teleblog.FormWebhookTextMarkup(rawMessage.Text, rawMessage.Entities)
					if err != nil {
						return err
					}
				} else if len(rawMessage.CaptionEntities) > 0 {
					comment.TextWithMarkup, err = teleblog.FormWebhookTextMarkup(rawMessage.Caption, rawMessage.CaptionEntities)
					if err != nil {
						return err
					}
				} else {
					comment.TextWithMarkup = strings.ReplaceAll(
						html.EscapeString(rawMessage.Text),
						"\n",
						"<br>",
					)
				}
			}
		}

		// # Add quote
		for _, comment := range comments {
			if comment.TgReplyToMessageId <= 0 || comment.TgReplyToMessageId == post.TgMessageId {
				continue
			}

			for _, repliedTo := range comments {
				if repliedTo.TgMessageId != comment.TgReplyToMessageId {
					continue
				}

				comment.ReplyToComment = &repliedTo.CommentWithTextWithMarkup
				break
			}
		}

		seo := views.SeoMetadata{
			Title:       post.Title,
			Description: post.SeoDescription,
			Image:       "",
			Url:         fmt.Sprintf("%s%s", app.Settings().Meta.AppUrl, views.GetPostUrl(post.Post)),
			Type:        "article",
		}

		if seo.Title == "" {
			seo.Title = templu.RemoveNewLines(fmt.Sprintf("%.60s", post.Text))
		}

		if seo.Description == "" {
			seo.Description = templu.RemoveNewLines(fmt.Sprintf("%.160s", post.Text))
		}

		if len(post.Media) > 0 {
			seo.Image = fmt.Sprintf("%s%s", app.Settings().Meta.AppUrl, post.Media[0])
		}

		// ## Header
		header := partials.HeaderData{
			LogoUrl: teleblog.ImagePath(
				siteConfigCollection,
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

		component := views.PostPage(
			views.BaseLayoutData{
				Seo:                    seo,
				YandexMetrikaCounter:   siteConfig.YandexMetrikaCounter,
				GoogleAnalyticsCounter: siteConfig.GoogleAnalyticsCounter,
				PrimaryColor:           siteConfig.PrimaryColor,
				BgImage: teleblog.ImagePath(
					siteConfigCollection,
					&siteConfig.BaseModel,
					siteConfig.BgImage,
				),
				CustomCss: siteConfig.CustomCss,
				FavIcon: teleblog.ImagePath(
					siteConfigCollection,
					&siteConfig.BaseModel,
					siteConfig.Favicon,
				),
			},
			views.PostPageData{
				Header: header,
				Footer: partials.FooterData{
					Text: siteConfig.Footer,
				},
			},
			chat,
			post,
			comments,
		)

		return component.Render(c.Request().Context(), c.Response().Writer)
	})
}
