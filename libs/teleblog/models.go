package teleblog

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

// # User

var _ models.Model = (*User)(nil)

type User struct {
	models.BaseModel

	Username     string `json:"username" db:"username"`
	Email        string `json:"email" db:"email"`
	Verified     bool   `json:"verified" db:"verified"`
	Name         string `json:"name" db:"name"`
	PasswordHash string `json:"-" db:"passwordHash"`
	TokenKey     string `db:"tokenKey" json:"-"`

	TgUserId   int64  `json:"tgUserId" db:"tg_user_id"`
	TgUsername string `json:"tgUsername" db:"tg_username"`
}

func (m *User) TableName() string {
	return "users"
}

func UserQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&User{})
}

// # Verification Token

var _ models.Model = (*TgVerificationToken)(nil)

type TgVerificationToken struct {
	models.BaseModel

	UserId   string `json:"userId" db:"user_id"`
	Value    string `json:"value" db:"value"`
	Verified bool   `json:"verified" db:"verified"`
}

func (m *TgVerificationToken) TableName() string {
	return "tg_verification_token"
}

func TgVerificationTokenQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&TgVerificationToken{})
}

// # Chat

var _ models.Model = (*Chat)(nil)

type Chat struct {
	models.BaseModel

	UserId       string `json:"userId" db:"user_id"`
	LinkedChatId string `json:"linkedChatId" db:"linked_chat_id"`

	TgUsername     string `json:"tgUsername" db:"tg_username"`
	TgChatId       int64  `json:"tgChatId" db:"tg_chat_id"`
	TgType         string `json:"tgType" db:"tg_type"` //  "private" | "group" | "supergroup" | "channel" | "privatechannel"
	TgLinkedChatId int64  `json:"tgLinkedChatId" db:"tg_linked_chat_id"`
}

func (m *Chat) TableName() string {
	return "chat"
}

func ChatQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Chat{})
}

// # Post

var _ models.Model = (*Post)(nil)

type Post struct {
	models.BaseModel

	ChatId             string `json:"chatId" db:"chat_id"`
	IsTgMessage        bool   `json:"isTgMessage" db:"is_tg_message"`
	IsTgHistoryMessage bool   `json:"isTgHistoryMessage" db:"is_tg_history_message"`

	Text string `json:"text" db:"text"`

	TgMessageId      int           `json:"tgMessageId" db:"tg_post_id"`
	TgGroupMessageId int           `json:"tgGroupMessageId" db:"tg_group_message_id"`
	TgMessageRaw     types.JsonMap `json:"tgMessageRaw" db:"tg_message_raw"`

	Media types.JsonArray[string] `json:"media" db:"media"`

	AlbumID string `json:"albumId" db:"album_id"`

	Title          string `json:"title" db:"title"`
	Slug           string `json:"slug" db:"slug"`
	SeoDescription string `json:"seoDescription" db:"seo_description"`

	Unparsable bool `json:"unparsable" db:"unparsable"`
}

func (m *Post) TableName() string {
	return "post"
}

func PostQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Post{})
}

// # Comment

var _ models.Model = (*Comment)(nil)

type Comment struct {
	models.BaseModel

	ChatId             string `json:"chatId" db:"chat_id"`
	PostId             string `json:"postId" db:"post_id"`
	IsTgHistoryMessage bool   `json:"isTgHistoryMessage" db:"is_tg_history_message"`

	Text string `json:"text" db:"text"`

	TgMessageId        int           `json:"tgMessageId" db:"tg_comment_id"`
	TgMessageRaw       types.JsonMap `json:"tgMessageRaw" db:"tg_message_raw"`
	TgReplyToMessageId int           `json:"tgReplyToMessageId" db:"tg_reply_to_message_id"`
}

func (m *Comment) TableName() string {
	return "comment"
}

func CommentQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Comment{})
}

// # Tag

var _ models.Model = (*Tag)(nil)

type Tag struct {
	models.BaseModel

	Value string `json:"value" db:"value"`
}

func (m *Tag) TableName() string {
	return "tag"
}

func TagQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Tag{})
}

// # PostTag

var _ models.Model = (*PostTag)(nil)

type PostTag struct {
	models.BaseModel

	ChatId string `json:"chatId" db:"chat_id"`
	PostId string `json:"postId" db:"post_id"`
	TagId  string `json:"tagId" db:"tag_id"`
}

func (m *PostTag) TableName() string {
	return "post_tag"
}

func PostTagQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Tag{})
}

// # Config

var _ models.Model = (*Config)(nil)

type Config struct {
	models.BaseModel

	Description    string `json:"description" db:"description"`
	SeoTitle       string `json:"seoTitle" db:"seo_title"`
	SeoDescription string `json:"seoDescription" db:"seo_description"`
	SeoImage       string `json:"seoImage" db:"seo_image"`
	SeoUrl         string `json:"seoUrl" db:"seo_url"`
	LogoUrl        string `json:"logoUrl" db:"logo_url"`
	LogoAlt        string `json:"logoAlt" db:"logo_alt"`
	Footer         string `json:"footer" db:"footer"`

	GoogleAnalyticsCounter string `json:"googleAnalyticsCounter" db:"google_analytics_counter"`
	YandexMetrikaCounter   string `json:"yandexMetrikaCounter" db:"yandex_metrika_counter"`

	PrimaryColor string `json:"mainColor" db:"main_color"`
	BgImage      string `json:"bgImage" db:"bg_image"`
	CustomCss    string `json:"customCss" db:"custom_css"`
	Favicon      string `json:"favicon" db:"favicon"`
}

func (m *Config) TableName() string {
	return "config"
}

func ConfigQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Config{})
}

func Configcollection(dao *daos.Dao) (*models.Collection, error) {
	config := &Config{}
	return dao.FindCollectionByNameOrId(config.TableName())
}

// # MenuItem

var _ models.Model = (*MenuItem)(nil)

type MenuItem struct {
	models.BaseModel

	Name     string `json:"name" db:"name"`
	Url      string `json:"url" db:"url"`
	Position int    `json:"position" db:"position"`
}

func (m *MenuItem) TableName() string {
	return "menu_item"
}

func MenuItemQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&MenuItem{})
}
