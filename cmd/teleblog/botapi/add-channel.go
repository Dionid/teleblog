package botapi

import (
	"fmt"
	"strings"

	"github.com/Dionid/teleblog/libs/teleblog"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"gopkg.in/telebot.v4"
)

func AddChannelCommand(b *telebot.Bot, app *pocketbase.PocketBase) {
	b.Handle("/"+ADD_CHANNEL_COMMAND_NAME, func(c telebot.Context) error {
		user := &teleblog.User{}

		err := teleblog.UserQuery(app.Dao()).
			AndWhere(dbx.HashExp{"tg_user_id": c.Sender().ID}).
			Limit(1).
			One(user)

		if err != nil {
			app.Logger().Error("!!! Error while getting user", "error: ", err)
			return c.Reply("You are not verified.")
		}

		tags := c.Args() // list of arguments splitted by a space

		if len(tags) == 0 {
			return c.Reply(fmt.Sprintf("You must provide channel name (e.g. /%s @YOUR_CHANNEL_NAME).", ADD_CHANNEL_COMMAND_NAME))
		}

		if len(tags) > 1 {
			return c.Reply("You can add only 1 channel at a time.")
		}

		channelUsername := tags[0]

		tgChannel, err := b.ChatByUsername(channelUsername)
		if err != nil {
			return c.Reply("No channel like this found.")
		}

		if tgChannel.Type != telebot.ChatChannel && tgChannel.Type != telebot.ChatChannelPrivate {
			return c.Reply("This is not a channel.")
		}

		channelMember, err := b.ChatMemberOf(tgChannel, c.Sender())
		if err != nil {
			return c.Reply("You cant add not your channels.")
		}

		if channelMember.Role != telebot.Administrator && channelMember.Role != telebot.Creator {
			return c.Reply("You are not the administrator of the channel.")
		}

		channel := &teleblog.Chat{}

		err = teleblog.ChatQuery(app.Dao()).
			AndWhere(dbx.HashExp{"tg_chat_id": tgChannel.ID, "user_id": user.Id}).
			Limit(1).
			One(channel)
		if err != nil {
			if !strings.Contains(err.Error(), "no rows in result set") {
				return err
			}

			newChannel := teleblog.Chat{
				UserId:       user.Id,
				LinkedChatId: "",

				TgUsername:     tgChannel.Username,
				TgChatId:       tgChannel.ID,
				TgType:         string(tgChannel.Type),
				TgLinkedChatId: tgChannel.LinkedChatID,
			}

			if err := app.Dao().Save(&newChannel); err != nil {
				return err
			}

			channel = &newChannel
		}

		// # Add linked chat
		if tgChannel.LinkedChatID > 0 {
			linkedGroup, err := b.ChatByID(tgChannel.LinkedChatID)
			if err != nil {
				return c.Reply("No channel chat like this found.")
			}

			channelsChat := &teleblog.Chat{}

			err = teleblog.ChatQuery(app.Dao()).
				AndWhere(dbx.HashExp{"tg_chat_id": linkedGroup.ID, "user_id": user.Id}).
				Limit(1).
				One(channelsChat)
			if err != nil {
				if !strings.Contains(err.Error(), "no rows in result set") {
					return err
				}

				newChannelsChat := teleblog.Chat{
					UserId:       user.Id,
					LinkedChatId: channel.Id,

					TgUsername:     linkedGroup.Username,
					TgChatId:       linkedGroup.ID,
					TgType:         string(linkedGroup.Type),
					TgLinkedChatId: linkedGroup.LinkedChatID,
				}

				if err := app.Dao().Save(&newChannelsChat); err != nil {
					return err
				}
			}
		}

		return c.Reply("Channel and linked group are successfully added.")
	})
}
