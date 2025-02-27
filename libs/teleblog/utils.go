package teleblog

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/telebot.v3"
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
