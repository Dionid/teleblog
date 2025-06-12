package botapi

import (
	"fmt"
	"time"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"gopkg.in/telebot.v4"
)

func VerifyTokenCommand(b *telebot.Bot, app *pocketbase.PocketBase) {
	b.Handle("/"+VERIFY_TOKEN_COMMAND_NAME, func(c telebot.Context) error {
		tags := c.Args() // list of arguments splitted by a space

		if len(tags) == 0 {
			return c.Reply(fmt.Sprintf("You must provide token (e.g. /%s TOKEN).", VERIFY_TOKEN_COMMAND_NAME))
		}

		if len(tags) > 1 {
			return c.Reply("Send only one token.")
		}

		receivedToken := tags[0]

		token := &teleblog.TgVerificationToken{}

		err := teleblog.TgVerificationTokenQuery(app.Dao()).
			AndWhere(dbx.HashExp{"value": receivedToken}).
			Limit(1).
			One(token)

		if err != nil {
			return c.Reply("Token not found.")
		}

		if token.Verified {
			return c.Reply("Token already verified.")
		}

		if token.Created.Time().Add(15 * time.Minute).Before(time.Now()) {
			return c.Reply("Token expired.")
		}

		user := &teleblog.User{}

		err = teleblog.UserQuery(app.Dao()).
			AndWhere(dbx.HashExp{"id": token.UserId}).
			Limit(1).
			One(user)

		if err != nil {
			return c.Reply("User not found.")
		}

		token.Verified = true
		if err := app.Dao().Save(token); err != nil {
			return err
		}

		user.TgUserId = c.Sender().ID
		user.TgUsername = c.Sender().Username
		if err := app.Dao().Save(user); err != nil {
			return err
		}

		return c.Reply("Successfully verified")
	})
}
