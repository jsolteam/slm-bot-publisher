package discord

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"slm-bot-publisher/internal/core/model"
	"slm-bot-publisher/internal/lib/storage"
	"slm-bot-publisher/logging"
	"strings"
)

type BotDiscord struct {
	Sessions      map[string]*discordgo.Session
	TelegramToken string
}

func NewDiscordBot(storage *storage.Storage, tgToken string) *BotDiscord {
	sessions := make(map[string]*discordgo.Session)

	for _, streamer := range storage.Streamers {
		dg, err := discordgo.New("Bot " + streamer.DiscordBotToken)
		if err != nil {
			logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка создания сессии Discord бота для %s: %v", streamer.Name, err))
		}

		err = dg.Open()
		if err != nil {
			logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка подключения Discord бота для %s: %v", streamer.Name, err))
		}

		sessions[streamer.Name] = dg
		logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Создана сессия Discord для %s", streamer.Name))
	}

	return &BotDiscord{Sessions: sessions, TelegramToken: tgToken}
}

func (d *BotDiscord) SendMessageToDiscord(streamer *model.Streamer, message string, photos [][]byte) {
	for _, discordChannel := range streamer.DiscordChannels {
		session := d.Sessions[streamer.Name]
		var prefix string
		if strings.HasPrefix(discordChannel.Prefix, "@") {
			prefix = discordChannel.Prefix
		} else {
			prefix = fmt.Sprintf("<@&%s>", discordChannel.Prefix)
		}

		files := []*discordgo.File{}

		for idxPhoto, photo := range photos {
			if len(photo) > 0 {
				files = append(files, &discordgo.File{
					Name:   fmt.Sprintf("photo_%d.jpg", idxPhoto),
					Reader: bytes.NewReader(photo),
				})
			}
		}

		_, err := session.ChannelMessageSendComplex(discordChannel.ChannelID, &discordgo.MessageSend{
			Content: prefix + " " + message,
			Files:   files,
		})
		if err != nil {
			logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка отправки сообщения на Discord канал %s: %v", discordChannel.ChannelID, err))
		}
		logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Произошла отправка сообщения стримера %s в канал %s", streamer.Name, discordChannel.ChannelID))
	}
}
