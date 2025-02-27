package teleblog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/telebot.v3"
)

type HistoryMessageTextEntity struct {
	Type telebot.EntityType `json:"type"`
	Text string             `json:"text"`
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
	Text              interface{}              `json:"text"` // Can be string or array of objects
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

type HistoryMessageReaction struct {
	Type  string `json:"type"` // emoji
	Count int    `json:"count"`
	Emoji string `json:"emoji"`
}

type History struct {
	Id       int64            `json:"id"`
	Name     string           `json:"name"`
	Type     string           `json:"type"` // "public_channel" | "public_supergroup"
	Messages []HistoryMessage `json:"messages"`
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

	// List all files in the given folder path
	files, err := filepath.Glob(filepath.Join(folderPath, "*"))
	if err != nil {
		return nil, err
	}

	// Iterate through the files to categorize them
	for _, file := range files {
		// Check if the file is a directory
		if fi, err := os.Stat(file); err != nil || fi.IsDir() {
			continue
		}
		switch filepath.Dir(file) {
		case filepath.Join(folderPath, "photos"):
			historyExport.Photos = append(historyExport.Photos, file)
		case filepath.Join(folderPath, "files"):
			historyExport.Files = append(historyExport.Files, file)
		case filepath.Join(folderPath, "video_files"):
			historyExport.VideoFiles = append(historyExport.VideoFiles, file)
		case filepath.Join(folderPath, "voice_messages"):
			historyExport.VoiceMessages = append(historyExport.VoiceMessages, file)
		case filepath.Join(folderPath, "round_video_messages"):
			historyExport.RoundVideoMessages = append(historyExport.RoundVideoMessages, file)
		case filepath.Join(folderPath, "stickers"):
			historyExport.Stickers = append(historyExport.Stickers, file)
		}
	}

	// Assuming result.json is directly in the folder path
	resultJsonPath := filepath.Join(folderPath, "result.json")
	if _, err := os.Stat(resultJsonPath); err != nil {
		return nil, err
	}

	historyExport.ResultJson = resultJsonPath

	return &historyExport, nil
}

func (h *History) GetChatTgId() (int64, error) {
	return strconv.ParseInt(fmt.Sprintf("-100%d", h.Id), 10, 64)
}
