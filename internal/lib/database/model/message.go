package modeldb

type Message struct {
	ID                   uint   `gorm:"primaryKey"`
	MainPost             bool   `gorm:"not null"`
	ChannelID            string `gorm:"not null"`
	TelegramMsgID        int    `gorm:"not null"`
	DiscordMsgID         string `gorm:"not null"`
	TelegramAttachmentID string `gorm:"default:null"`
	DiscordAttachmentID  string `gorm:"default:null"`
}
