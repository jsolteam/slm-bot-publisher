package model

type Streamer struct {
	Name              string
	TelegramChannelID int64
	DiscordBotToken   string
	DiscordChannels   []DiscordChannel
}
