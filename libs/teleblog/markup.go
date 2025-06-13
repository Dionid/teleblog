package teleblog

import (
	"html"
	"slices"
	"sort"
	"strings"
	"unicode/utf16"

	"gopkg.in/telebot.v4"
)

type MarkupByPosition struct {
	Offset   int
	Priority int
	IsOpen   bool
	Tag      []rune
}

func FormHistoryRawTextWithMarkup(markup HistoryMessageText) string {
	text := ""

	for _, entity := range markup.Items {
		switch entity.Type {
		case telebot.EntityItalic:
			text += "<i class='inline'>" + entity.Text + "</i>"
		case telebot.EntityBold:
			text += "<b class='inline'>" + entity.Text + "</b>"
		case telebot.EntityURL:
			link := entity.Text
			if strings.Contains(link, "://") == false {
				link = "http://" + link
			}
			text += "<a target='_blank' href='https://" + link + "' class='inline c-link'>" + entity.Text + "</a>"
		case "link":
			text += "<a target='_blank' class='inline c-link' href='https://" + entity.Text + "'>" + entity.Text + "</a>"
		case telebot.EntityHashtag:
			tag, err := CorrectTagValue(entity.Text)
			if err != nil {
				continue
			}
			text += "<a class='inline c-link' href='?tag=" + tag + "'>" + entity.Text + "</a>"
		case telebot.EntityTextLink:
			text += "<a target='_blank' class='inline c-link' href='" + entity.Href + "'>" + entity.Text + "</a>"
		case telebot.EntityMention:
			mention := strings.TrimPrefix(entity.Text, "@")
			text += "<a target='_blank' href='https://t.me/" + mention + "' class='inline c-link'>" + entity.Text + "</a>"
		default:
			// escapedText := html.EscapeString(entity.Text)
			text += html.EscapeString(entity.Text)
		}
	}

	text = strings.ReplaceAll(text, "\\n", "<br>")
	text = strings.ReplaceAll(text, "\n", "<br>")

	return text
}

func HistoryTextEntitiesWithToTextWithMarkup(markup []HistoryMessageTextEntity) string {
	text := ""

	for _, entity := range markup {
		switch entity.Type {
		case telebot.EntityItalic:
			text += "<i class='inline'>" + entity.Text + "</i>"
		case telebot.EntityBold:
			text += "<b class='inline'>" + entity.Text + "</b>"
		case telebot.EntityURL:
			link := entity.Text
			if strings.Contains(link, "://") == false {
				link = "http://" + link
			}
			text += "<a target='_blank' href='https://" + link + "' class='inline c-link'>" + entity.Text + "</a>"
		case "link":
			text += "<a target='_blank' class='inline c-link' href='https://" + entity.Text + "'>" + entity.Text + "</a>"
		case telebot.EntityHashtag:
			tag, err := CorrectTagValue(entity.Text)
			if err != nil {
				continue
			}
			text += "<a class='inline c-link' href='?tag=" + tag + "'>" + entity.Text + "</a>"
		case telebot.EntityTextLink:
			text += "<a target='_blank' class='inline c-link' href='https://" + entity.Text + "'>" + entity.Text + "</a>"
		case telebot.EntityMention:
			text += "<a target='_blank' href='https://t.me/" + entity.Text + "' class='inline c-link'>" + entity.Text + "</a>"
		default:
			escapedText := html.EscapeString(entity.Text)
			newLineText := strings.ReplaceAll(escapedText, "\n", "<br>")
			text += newLineText
		}
	}

	return text
}

func FormWebhookTextMarkup(srcText string, entities telebot.Entities) (string, error) {
	escapedText := html.EscapeString(srcText)
	text := utf16.Encode([]rune(escapedText))

	var markUpByPosition []MarkupByPosition

	for i, entity := range entities {
		switch entity.Type {
		case telebot.EntityItalic:
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset, Tag: []rune("<i class='inline'>"), Priority: i, IsOpen: true})
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset + entity.Length, Tag: []rune("</i>"), Priority: i, IsOpen: false})
		case telebot.EntityBold:
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset, Tag: []rune("<b class='inline'>"), Priority: i, IsOpen: true})
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset + entity.Length, Tag: []rune("</b>"), Priority: i, IsOpen: false})
		case telebot.EntityHashtag:
			tag, err := CorrectTagValue(string(utf16.Decode(text[entity.Offset : entity.Offset+entity.Length])))
			if err != nil {
				continue
			}
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset, Tag: []rune("<a href='?tag=" + tag + "' class='inline c-link'>"), Priority: i, IsOpen: true})
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset + entity.Length, Tag: []rune("</a>"), Priority: i, IsOpen: false})
		case telebot.EntityURL:
			link := string(utf16.Decode(text[entity.Offset : entity.Offset+entity.Length]))
			if strings.Contains(link, "://") == false {
				link = "http://" + link
			}
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset, Tag: []rune("<a target='_blank' href='" + link + "' class='inline c-link'>"), Priority: i, IsOpen: true})
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset + entity.Length, Tag: []rune("</a>"), Priority: i, IsOpen: false})
		case telebot.EntityTextLink:
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset, Tag: []rune("<a target='_blank' class='inline c-link' href='" + entity.URL + "'>"), Priority: i, IsOpen: true})
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset + entity.Length, Tag: []rune("</a>"), Priority: i, IsOpen: false})
		case telebot.EntityMention:
			link := string(utf16.Decode(text[entity.Offset+1 : entity.Offset+entity.Length]))
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset, Tag: []rune("<a target='_blank' href='https://t.me/" + link + "' class='inline c-link'>"), Priority: i, IsOpen: true})
			markUpByPosition = append(markUpByPosition, MarkupByPosition{Offset: entity.Offset + entity.Length, Tag: []rune("</a>"), Priority: i, IsOpen: false})
		default:
			continue
		}
	}

	sort.Slice(markUpByPosition, func(i, j int) bool {
		a := markUpByPosition[i]
		b := markUpByPosition[j]

		// # If they are on the same place
		if a.Offset == b.Offset {
			// # If it is closing tag, then more prior must be first
			if a.IsOpen == false && b.IsOpen == false {
				return a.Priority < b.Priority
			}

			return a.Priority > b.Priority
		} else {
			return a.Offset > b.Offset
		}
	})

	for _, markup := range markUpByPosition {
		text = slices.Insert(text, markup.Offset, utf16.Encode(markup.Tag)...)
	}

	resultText := string(utf16.Decode(text))
	newLineResultText := strings.ReplaceAll(resultText, "\n", "<br>")

	return newLineResultText, nil
}
