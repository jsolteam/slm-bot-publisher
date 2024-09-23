package discord

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"slm-bot-publisher/internal/core/model"
	"slm-bot-publisher/internal/lib/database/handlers"
	modeldb "slm-bot-publisher/internal/lib/database/model"
	"slm-bot-publisher/internal/lib/storage"
	"slm-bot-publisher/logging"
	"strings"
)

type BotDiscord struct {
	SessionCreators map[string]func() (*discordgo.Session, error)
	TelegramToken   string
	DBHandlers      *handlers.DBHandlers
}

func NewDiscordBot(storage *storage.Storage, tgToken string, DBHandlers *handlers.DBHandlers) *BotDiscord {
	sessionCreators := make(map[string]func() (*discordgo.Session, error))

	for _, streamer := range storage.Streamers {
		sessionCreators[streamer.Name] = func(s *model.Streamer) func() (*discordgo.Session, error) {
			return func() (*discordgo.Session, error) {
				return createSession(s)
			}
		}(&streamer)
	}

	return &BotDiscord{SessionCreators: sessionCreators, TelegramToken: tgToken, DBHandlers: DBHandlers}
}

// createSession создает и открывает сессию Discord для стримера
func createSession(streamer *model.Streamer) (*discordgo.Session, error) {
	dg, err := discordgo.New("Bot " + streamer.DiscordBotToken)
	if err != nil {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка создания сессии Discord для %s: %v", streamer.Name, err))
		return nil, err
	}

	if err = dg.Open(); err != nil {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка подключения к Discord для %s: %v", streamer.Name, err))
		return nil, err
	}

	logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сессия Discord для %s создана и открыта", streamer.Name))
	return dg, nil
}

// sendWithSession - вспомогательная функция для взаимодействия с Discord с использованием сессии
func (d *BotDiscord) sendWithSession(streamer *model.Streamer, sendFunc func(*discordgo.Session) error) {
	sessionCreator, exists := d.SessionCreators[streamer.Name]
	if !exists {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Стример %s не найден", streamer.Name))
		return
	}

	session, err := sessionCreator()
	if err != nil {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка создания сессии для стримера %s: %v", streamer.Name, err))
		return
	}
	defer session.Close()

	if err = sendFunc(session); err != nil {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка отправки запроса для стримера %s: %v", streamer.Name, err))
	}
}

// SendMessageToDiscord отправляет сообщение с вложениями в Discord
func (d *BotDiscord) SendMessageToDiscord(streamer *model.Streamer, message string, attachments []*discordgo.File, messageModel []modeldb.Message) {
	filesData := make([][]byte, len(attachments))

	for i, attachment := range attachments {
		fileData, err := ioutil.ReadAll(attachment.Reader)
		if err != nil {
			logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка чтения файла: %v", err))
			return
		}
		filesData[i] = fileData
	}

	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		for _, discordChannel := range streamer.DiscordChannels {
			prefix := formatPrefix(discordChannel.Prefix)

			files := prepareFiles(attachments, filesData)

			message, err := session.ChannelMessageSendComplex(discordChannel.ChannelID, &discordgo.MessageSend{
				Content: prefix + " " + message,
				Files:   files,
			})

			if err != nil {
				return fmt.Errorf("ошибка отправки сообщения на канал %s: %v", discordChannel.ChannelID, err)
			}

			for idx, msg := range messageModel {
				messageDB := modeldb.Message{
					MainPost:      msg.MainPost,
					ChannelID:     discordChannel.ChannelID,
					TelegramMsgID: msg.TelegramMsgID,
					DiscordMsgID:  message.ID,
				}
				if msg.TelegramAttachmentID != "" && message.Attachments[idx] != nil {
					messageDB.TelegramAttachmentID = msg.TelegramAttachmentID
					messageDB.DiscordAttachmentID = message.Attachments[idx].ID
				}

				err := d.DBHandlers.MessageHandlers.CreateMessage(&messageDB)
				if err != nil {
					logging.Log("Database", logrus.InfoLevel, fmt.Sprintf("Ошибка сохранения сообщения %d в базу", messageDB.TelegramMsgID))
				}
			}
			logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сообщение от %s успешно отправлено в канал %s", streamer.Name, discordChannel.ChannelID))
		}
		return nil
	})
}

// SendRepostToDiscord отправляет репост в Discord
func (d *BotDiscord) SendRepostToDiscord(streamer *model.Streamer, repost model.DiscordRepost) {
	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		for _, discordChannel := range streamer.DiscordChannels {
			embed := &discordgo.MessageEmbed{
				Author: &discordgo.MessageEmbedAuthor{
					Name:    fmt.Sprintf("Переслано из %s", repost.ChannelName),
					IconURL: repost.ChannelAvatar,
					URL:     repost.RepostLink,
				},
				Description: repost.MessageContent,
				Color:       1796358,
				Image: &discordgo.MessageEmbedImage{
					URL: repost.PhotoLink,
				},
				URL: repost.RepostLink,
			}

			_, err := session.ChannelMessageSendEmbed(discordChannel.ChannelID, embed)
			if err != nil {
				return fmt.Errorf("ошибка отправки сообщения на канал %s: %v", discordChannel.ChannelID, err)
			}
			logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Репост от %s успешно отправлен в канал %s", streamer.Name, discordChannel.ChannelID))
		}
		return nil
	})
}

// DeleteMessageFromDiscord удаляет сообщение из Discord
func (d *BotDiscord) DeleteMessageFromDiscord(streamer *model.Streamer, channelID, msgID string) {
	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		err := session.ChannelMessageDelete(channelID, msgID)
		if err != nil {
			return fmt.Errorf("ошибка удаления сообщения с ID %s в канале %s: %v", msgID, channelID, err)
		}
		logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сообщение с ID %s успешно удалено из канала %s", msgID, channelID))
		return nil
	})
}

// formatPrefix возвращает форматированный префикс для уведомлений
func formatPrefix(prefix string) string {
	if strings.HasPrefix(prefix, "@") {
		return prefix // Прямое использование, если это @everyone или @here
	}
	return fmt.Sprintf("<@&%s>", prefix) // Использование ID роли
}

// prepareFiles подготавливает файлы для отправки
func prepareFiles(attachments []*discordgo.File, filesData [][]byte) []*discordgo.File {
	files := make([]*discordgo.File, len(attachments))
	for i, fileData := range filesData {
		files[i] = &discordgo.File{
			Name:   attachments[i].Name,
			Reader: bytes.NewReader(fileData),
		}
	}
	return files
}
