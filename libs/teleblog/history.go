package teleblog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/telebot.v4"
)

type HistoryMessageTextEntity struct {
	Type telebot.EntityType `json:"type"`
	Text string             `json:"text"`
}

const (
	EntityPlainText telebot.EntityType = "plain_text"
)

type HistoryMessageTextItem struct {
	Type       telebot.EntityType `json:"type"`                  // "text" | "bold" | "italic" | "link" | "hashtag" | "mention" | "text_link"
	Text       string             `json:"text"`                  // The text content of the entity
	Href       string             `json:"href,omitempty"`        // For "text_link" entities, the URL they point to
	DocumentId string             `json:"document_id,omitempty"` // For "document" entities, the ID of the document
}

// Can be string or array of objects
type HistoryMessageText struct {
	Items []HistoryMessageTextItem `json:"entities"`
}

// HistoryMessageTextConverted is used to convert the JSON structure
type HistoryMessageTextConverted struct {
	Items []HistoryMessageTextItem `json:"entities"`
}

// HistoryMessage represents a single message in the chat history
// It includes all fields from result.json and additional fields like reactions
// and media types.

func (h *HistoryMessageText) UnmarshalJSON(data []byte) error {
	// check if it is a string
	if len(data) > 0 && data[0] == '"' && data[len(data)-1] == '"' {
		text := string(data[1 : len(data)-1])

		h.Items = []HistoryMessageTextItem{
			{
				Type: EntityPlainText,
				Text: text,
			},
		}

		return nil
	}

	converted := HistoryMessageTextConverted{}
	if err := json.Unmarshal(data, &converted); err == nil {
		h.Items = converted.Items
		return nil
	}

	var entities []any
	if err := json.Unmarshal(data, &entities); err != nil {
		return fmt.Errorf("unmarshal HistoryMessageText: %w", err)
	}

	h.Items = []HistoryMessageTextItem{}

	// Check if "text" is a string or an array of objects
	for _, entity := range entities {
		switch e := entity.(type) {
		case string:
			// If it's a string, create a text entity
			h.Items = append(h.Items, HistoryMessageTextItem{
				Type: EntityPlainText,
				Text: e,
			})
		case map[string]any:
			// If it's a map, check for "type" and "text" keys
			if entityType, ok := e["type"].(string); ok {
				switch entityType {
				case "text_link":
					if text, ok := e["text"].(string); ok {
						href, ok := e["href"].(string)
						if !ok {
							return fmt.Errorf("missing or invalid 'href' field in text_href entity")
						}

						h.Items = append(h.Items, HistoryMessageTextItem{
							Type: telebot.EntityType(entityType),
							Text: text,
							Href: href,
						})
					} else {
						fmt.Printf("missing or invalid 'text' field in entity of type '%s'", entityType)
					}
				default:
					if text, ok := e["text"].(string); ok {
						h.Items = append(h.Items, HistoryMessageTextItem{
							Type: telebot.EntityType(entityType),
							Text: text,
						})
					} else {
						fmt.Printf("Warning: missing or invalid 'text' field in entity of type '%s'\n", entityType)
					}
				}
			}
		}
	}

	return nil
}

func (h *HistoryMessageText) MarshalJSON() ([]byte, error) {
	if len(h.Items) == 0 {
		return json.Marshal("")
	}

	// If all items are text, return as a string
	allText := true
	for _, item := range h.Items {
		if item.Type != "text" {
			allText = false
			break
		}
	}

	if allText {
		text := ""
		for _, item := range h.Items {
			text += item.Text
		}
		return json.Marshal(text)
	}

	toMarshall := make([]any, len(h.Items))
	for i, item := range h.Items {
		if item.Type == EntityPlainText {
			toMarshall[i] = item.Text
		} else {
			ent := map[string]any{
				"type": item.Type,
				"text": item.Text,
			}
			if item.Href != "" {
				ent["href"] = item.Href
			}
			if item.DocumentId != "" {
				ent["document_id"] = item.DocumentId
			}
			toMarshall[i] = ent
		}
	}
	if len(toMarshall) == 1 && toMarshall[0] == "" {
		// If there's only one item and it's an empty string, return as an empty string
		return json.Marshal("")
	}

	return json.Marshal(toMarshall)
}

type HistoryMessageReaction struct {
	Type  string `json:"type"` // emoji
	Count int    `json:"count"`
	Emoji string `json:"emoji"`
}

type HistoryMessage struct {
	Id               int                        `json:"id"`
	Type             string                     `json:"type"` // service | message
	Date             string                     `json:"date"`
	DateUnix         string                     `json:"date_unixtime"`
	Edited           string                     `json:"edited"`
	EditedUnix       string                     `json:"edited_unixtime"`
	From             string                     `json:"from"`
	FromId           string                     `json:"from_id"`
	TextEntities     []HistoryMessageTextEntity `json:"text_entities"`
	File             *string                    `json:"file"`
	Photo            *string                    `json:"photo"`
	ReplyToMessageId int                        `json:"reply_to_message_id"`
	ForwardedFrom    *string                    `json:"forwarded_from"`
	// Additional fields from result.json
	Actor             string                   `json:"actor"`
	ActorId           string                   `json:"actor_id"`
	Action            string                   `json:"action"`
	Title             string                   `json:"title"`
	Text              HistoryMessageText       `json:"text"` // Can be string or array of objects
	FileName          string                   `json:"file_name"`
	FileSize          int                      `json:"file_size"`
	Thumbnail         string                   `json:"thumbnail"`
	ThumbnailFileSize int                      `json:"thumbnail_file_size"`
	MediaType         string                   `json:"media_type"` // video_file, voice_message, video_message
	MimeType          string                   `json:"mime_type"`
	DurationSeconds   int                      `json:"duration_seconds"`
	Width             int                      `json:"width"`
	Height            int                      `json:"height"`
	PhotoFileSize     int                      `json:"photo_file_size"`
	Reactions         []HistoryMessageReaction `json:"reactions"`
}

type History struct {
	Id       int64            `json:"id"`
	Name     string           `json:"name"`
	Type     string           `json:"type"` // "public_channel" | "public_supergroup" | "private_supergroup"
	Messages []HistoryMessage `json:"messages"`
}

func (h *History) GetChatTgId() (int64, error) {
	return strconv.ParseInt(fmt.Sprintf("-100%d", h.Id), 10, 64)
}

// HistoryZip represents the folder structure of a Telegram chat export
type HistoryExport struct {
	ResultJson         string   // result.json file
	Photos             []string // photos directory - contains jpg files
	Files              []string // files directory - contains various files (HEIC, etc)
	VideoFiles         []string // video_files directory - contains video files and stickers (webm)
	VoiceMessages      []string // voice_messages directory - contains voice messages (ogg)
	RoundVideoMessages []string // round_video_messages directory - contains round video messages (mp4)
	Stickers           []string // stickers directory - contains sticker files (tgs)
}

func FolderToHistoryExport(folderPath string) (*HistoryExport, error) {
	// Initialize an empty HistoryExport struct
	historyExport := HistoryExport{}

	// Walk through the directory tree
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Categorize files based on their directory
		switch filepath.Dir(path) {
		case filepath.Join(folderPath, "photos"):
			historyExport.Photos = append(historyExport.Photos, path)
		case filepath.Join(folderPath, "files"):
			historyExport.Files = append(historyExport.Files, path)
		case filepath.Join(folderPath, "video_files"):
			historyExport.VideoFiles = append(historyExport.VideoFiles, path)
		case filepath.Join(folderPath, "voice_messages"):
			historyExport.VoiceMessages = append(historyExport.VoiceMessages, path)
		case filepath.Join(folderPath, "round_video_messages"):
			historyExport.RoundVideoMessages = append(historyExport.RoundVideoMessages, path)
		case filepath.Join(folderPath, "stickers"):
			historyExport.Stickers = append(historyExport.Stickers, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Assuming result.json is directly in the folder path
	resultJsonPath := filepath.Join(folderPath, "result.json")
	if _, err := os.Stat(resultJsonPath); err != nil {
		return nil, err
	}

	historyExport.ResultJson = resultJsonPath

	return &historyExport, nil
}
