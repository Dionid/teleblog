package teleblog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf16"

	"gopkg.in/telebot.v4"
)

func CorrectTagValue(rawValue string) (string, error) {
	regex, err := regexp.Compile(`^(#[\w]+)`)
	if err != nil {
		return "", err
	}

	if rawValue[0] != '#' {
		return "", fmt.Errorf("Tag value must start with #")
	}

	value := strings.Replace(regex.FindString(rawValue), "#", "", -1)

	if value == "" {
		return "", fmt.Errorf("Tag value is empty")
	}

	return value, nil
}

func ExtractTagsFromPost(post Post) ([]string, error) {
	tags := []string{}

	jb, err := post.TgMessageRaw.MarshalJSON()
	if err != nil {
		return nil, err
	}

	if post.IsTgHistoryMessage {
		rawMessage := HistoryMessage{}

		err := json.Unmarshal(jb, &rawMessage)
		if err != nil {
			return nil, fmt.Errorf("ExtractTagsFromPost: (post_id: %s) unmarshal history message error: %w", post.Id, err)
		}

		for _, entity := range rawMessage.TextEntities {
			if entity.Type == telebot.EntityHashtag {
				value := strings.Replace(entity.Text, "#", "", -1)
				tags = append(tags, value)
			}
		}
	} else {
		rawMessage := telebot.Message{}

		err := json.Unmarshal(jb, &rawMessage)
		if err != nil {
			return nil, fmt.Errorf("ExtractTagsFromPost: (post_id: %s) realtime unmarshal message error: %w", post.Id, err)
		}

		text := utf16.Encode([]rune(post.Text))

		for _, entity := range rawMessage.Entities {
			if entity.Type == telebot.EntityHashtag {
				value, err := CorrectTagValue(string(utf16.Decode(text[entity.Offset : entity.Offset+entity.Length])))
				if err != nil {
					continue
				}

				tags = append(tags, value)
			}
		}
	}

	return tags, nil
}
