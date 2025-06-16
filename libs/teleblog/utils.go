package teleblog

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/telebot.v4"
)

func WriteJsonMessage(message *telebot.Message) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("message_%d.json", message.ID), jsonMessage, 0644)
	if err != nil {
		return err
	}

	return nil
}

func FormNegativeTgIdFromString(id string) (int64, error) {
	forwardFromTgId, err := strconv.ParseInt(
		fmt.Sprintf(
			"-100%s",
			strings.ReplaceAll(
				strings.ReplaceAll(
					id,
					"user",
					"",
				),
				"channel",
				"",
			),
		),
		10,
		64,
	)
	if err != nil {
		return 0, err
	}

	return forwardFromTgId, err
}

func FormNegativeTgIdFromInt(id int64) (int64, error) {
	forwardFromTgId, err := strconv.ParseInt(
		fmt.Sprintf(
			"-100%d",
			id,
		),
		10,
		64,
	)
	if err != nil {
		return 0, err
	}

	return forwardFromTgId, err
}
