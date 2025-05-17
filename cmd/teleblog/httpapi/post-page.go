package httpapi

import (
	"encoding/json"
	"fmt"

	"github.com/Dionid/teleblog/cmd/teleblog/httpapi/views"
	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/Dionid/teleblog/libs/templu"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/telebot.v3"
)

func PostPageHandler(e *core.ServeEvent, app core.App) {
	e.Router.GET("/post/:id", func(c echo.Context) error {
		postIdOrSlug := c.PathParam("id")

		post := views.PostPagePost{}
		err := teleblog.PostQuery(app.Dao()).Where(
			dbx.Or(
				dbx.HashExp{"id": postIdOrSlug},
				dbx.HashExp{"slug": postIdOrSlug},
			),
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

			post.TextWithMarkup = teleblog.FormHistoryTextWithMarkup(rawMessage.TextEntities)
		} else {
			rawMessage := telebot.Message{}

			err = json.Unmarshal(jb, &rawMessage)
			if err != nil {
				return err
			}

			post.TextWithMarkup, err = teleblog.FormWebhookTextMarkup(post.Text, rawMessage.Entities)
			if err != nil {
				return err
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
				return err
			}

			if comment.IsTgHistoryMessage {
				rawMessage := teleblog.HistoryMessage{}

				err = json.Unmarshal(jb, &rawMessage)
				if err != nil {
					return err
				}

				comment.AuthorTitle = rawMessage.From

				comment.TextWithMarkup = teleblog.FormHistoryTextWithMarkup(rawMessage.TextEntities)
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

				comment.TextWithMarkup, err = teleblog.FormWebhookTextMarkup(comment.Text, rawMessage.Entities)
				if err != nil {
					return err
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
			Url:         fmt.Sprintf("https://davidshekunts.ru%s", views.GetPostUrl(post.Post)),
			Type:        "article",
		}

		if seo.Title == "" {
			seo.Title = templu.RemoveNewLines(fmt.Sprintf("%.60s", post.Text))
		}

		if seo.Description == "" {
			seo.Description = templu.RemoveNewLines(fmt.Sprintf("%.160s", post.Text))
		}

		if len(post.Media) > 0 {
			seo.Image = fmt.Sprintf("https://davidshekunts.ru%s", post.Media[0])
		}

		component := views.PostPage(chat, post, comments, &seo)

		return component.Render(c.Request().Context(), c.Response().Writer)
	})
}
