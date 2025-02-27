package teleblog

import (
	"archive/zip"
	"fmt"
	"strconv"
	"strings"

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

// HistoryFolderStructure represents the folder structure of a Telegram chat export
type HistoryFolderStructure struct {
	ResultJson         string   // result.json file
	Photos             []string // photos directory - contains jpg files
	Files              []string // files directory - contains various files (HEIC, etc)
	VideoFiles         []string // video_files directory - contains video files and stickers (webm)
	VoiceMessages      []string // voice_messages directory - contains voice messages (ogg)
	RoundVideoMessages []string // round_video_messages directory - contains round video messages (mp4)
	Stickers           []string // stickers directory - contains sticker files (tgs)
}

// ParseZipIntoFolderStructure parses a zip reader into HistoryFolderStructure
func ParseZipIntoFolderStructure(zipReader *zip.Reader) (*HistoryFolderStructure, error) {
	structure := &HistoryFolderStructure{}

	for _, f := range zipReader.File {
		switch {
		case f.Name == "result.json":
			structure.ResultJson = f.Name
		case strings.HasPrefix(f.Name, "photos/"):
			structure.Photos = append(structure.Photos, f.Name)
		case strings.HasPrefix(f.Name, "files/"):
			structure.Files = append(structure.Files, f.Name)
		case strings.HasPrefix(f.Name, "video_files/"):
			structure.VideoFiles = append(structure.VideoFiles, f.Name)
		case strings.HasPrefix(f.Name, "voice_messages/"):
			structure.VoiceMessages = append(structure.VoiceMessages, f.Name)
		case strings.HasPrefix(f.Name, "round_video_messages/"):
			structure.RoundVideoMessages = append(structure.RoundVideoMessages, f.Name)
		case strings.HasPrefix(f.Name, "stickers/"):
			structure.Stickers = append(structure.Stickers, f.Name)
		}
	}

	if structure.ResultJson == "" {
		return nil, fmt.Errorf("no result.json found in zip file")
	}

	return structure, nil
}

func (h *History) GetChatTgId() (int64, error) {
	return strconv.ParseInt(fmt.Sprintf("-100%d", h.Id), 10, 64)
}
