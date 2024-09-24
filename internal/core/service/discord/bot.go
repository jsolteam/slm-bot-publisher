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

const (
	ThreadName          = "Комментарии"
	AutoArchiveDuration = 60
	FirstMessageContent = "Пожалуйста, соблюдайте правила общения в комментариях!"
)

func NewDiscordBot(storage *storage.Storage, tgToken string, DBHandlers *handlers.DBHandlers) *BotDiscord {
	sessionCreators := make(map[string]func() (*discordgo.Session, error))

	for _, streamer := range storage.Streamers {
		sessionCreators[streamer.Name] = func(s *model.Streamer) func() (*discordgo.Session, error) {
			return func() (*discordgo.Session, error) {
				return createSession(s)
			}
		}(&streamer)
	}

	return &BotDiscord{
		SessionCreators: sessionCreators,
		TelegramToken:   tgToken,
		DBHandlers:      DBHandlers,
	}
}

// createSession - создает и открывает сессию Discord для стримера
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

// sendMessage - отправляет сообщение в канал Discord с вложениями
func (d *BotDiscord) sendMessage(session *discordgo.Session, channelID, content string, files []*discordgo.File, postLink string) (*discordgo.Message, error) {
	sentMessage, err := session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Files:   files,
		Embed: &discordgo.MessageEmbed{
			Description: "Оригинальный пост: " + postLink,
		},
	})
	if err != nil {
		return nil, err
	}

	// Создаём ветку (тред) с названием "Комментарии" с автоархивом через 1 час
	thread, err := session.MessageThreadStart(channelID, sentMessage.ID, ThreadName, AutoArchiveDuration)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания ветки: %v", err)
	}

	// Отправляем первое сообщение в ветку с правилами общения
	_, err = session.ChannelMessageSend(thread.ID, FirstMessageContent)
	if err != nil {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка отправки сообщения с правилами в ветку: %v", err))
	}

	// Удаляем системное сообщение о создании ветки
	err = d.deleteSystemThreadMessage(session, channelID)
	if err != nil {
		logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка удаления системного сообщения о создании ветки: %v", err))
	}

	return sentMessage, nil
}

// deleteSystemThreadMessage - находит и удаляет системное сообщение о создании ветки
func (d *BotDiscord) deleteSystemThreadMessage(session *discordgo.Session, channelID string) error {
	// Получаем последние сообщения канала
	messages, err := session.ChannelMessages(channelID, 10, "", "", "")
	if err != nil {
		return fmt.Errorf("ошибка получения сообщений канала: %v", err)
	}

	for _, msg := range messages {
		if msg.Type == discordgo.MessageTypeThreadCreated {
			err := session.ChannelMessageDelete(channelID, msg.ID)
			if err != nil {
				return fmt.Errorf("ошибка удаления сообщения с ID %s: %v", msg.ID, err)
			}
			break
		}
	}
	return nil
}

// SendMessageToDiscord - отправляет сообщение с вложениями в Discord
func (d *BotDiscord) SendMessageToDiscord(streamer *model.Streamer, message string, attachments []*discordgo.File, messageModel []modeldb.Message, postLink string) {
	filesData := readFilesData(attachments)
	if filesData == nil {
		return
	}

	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		for _, discordChannel := range streamer.DiscordChannels {
			content := formatPrefix(discordChannel.Prefix) + " " + message
			files := prepareFiles(attachments, filesData)

			sentMessage, err := d.sendMessage(session, discordChannel.ChannelID, content, files, postLink)
			if err != nil {
				return fmt.Errorf("ошибка отправки сообщения на канал %s: %v", discordChannel.ChannelID, err)
			}

			d.saveMessagesToDB(sentMessage, discordChannel.ChannelID, messageModel)
			logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сообщения от %s успешно отправлено в канал %s", streamer.Name, discordChannel.ChannelID))
		}
		return nil
	})
}

// readFilesData - читает данные файлов из вложений
func readFilesData(attachments []*discordgo.File) [][]byte {
	filesData := make([][]byte, len(attachments))
	for i, attachment := range attachments {
		fileData, err := ioutil.ReadAll(attachment.Reader)
		if err != nil {
			logging.Log("Discord", logrus.ErrorLevel, fmt.Sprintf("Ошибка чтения файла: %v", err))
			return nil
		}
		filesData[i] = fileData
	}
	return filesData
}

// saveMessagesToDB - сохраняет отправленные сообщения в базе данных
func (d *BotDiscord) saveMessagesToDB(sentMessage *discordgo.Message, channelID string, messageModel []modeldb.Message) {
	for idx, msg := range messageModel {
		messageDB := modeldb.Message{
			MainPost:      msg.MainPost,
			ChannelID:     channelID,
			TelegramMsgID: msg.TelegramMsgID,
			DiscordMsgID:  sentMessage.ID,
		}
		if msg.TelegramAttachmentID != "" && sentMessage.Attachments[idx] != nil {
			messageDB.TelegramAttachmentID = msg.TelegramAttachmentID
			messageDB.DiscordAttachmentID = sentMessage.Attachments[idx].ID
		}

		err := d.DBHandlers.MessageHandlers.CreateMessage(&messageDB)
		if err != nil {
			logging.Log("Database", logrus.ErrorLevel, fmt.Sprintf("Ошибка сохранения сообщения %d в базу", messageDB.TelegramMsgID))
		}
	}
}

// prepareFiles - подготавливает файлы для отправки
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

// SendRepostToDiscord - отправляет репост в Discord
func (d *BotDiscord) SendRepostToDiscord(streamer *model.Streamer, repost model.DiscordRepost, messageModel modeldb.Message) {
	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		embed := buildRepostEmbed(repost)
		for _, discordChannel := range streamer.DiscordChannels {
			sentMessage, err := session.ChannelMessageSendEmbed(discordChannel.ChannelID, embed)
			if err != nil {
				return fmt.Errorf("ошибка отправки сообщения на канал %s: %v", discordChannel.ChannelID, err)
			}

			d.saveMessagesToDB(sentMessage, discordChannel.ChannelID, []modeldb.Message{messageModel})
			logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сообщения от %s успешно отправлено в канал %s", streamer.Name, discordChannel.ChannelID))
		}
		return nil
	})
}

// buildRepostEmbed - создает встраиваемое сообщение (embed) для репоста
func buildRepostEmbed(repost model.DiscordRepost) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
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
}

// EditMessageOnDiscord - редактирует сообщение в Discord
func (d *BotDiscord) EditMessageOnDiscord(streamer *model.Streamer, channel *model.DiscordChannel, message, msgID string) {
	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		content := formatPrefix(channel.Prefix) + " " + message
		_, err := session.ChannelMessageEdit(channel.ChannelID, msgID, content)
		if err != nil {
			return fmt.Errorf("ошибка изменения сообщения на канале %s: %v", channel.ChannelID, err)
		}
		logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сообщение %s успешно изменено в канале %s", msgID, channel.ChannelID))
		return nil
	})
}

// DeleteMessageFromDiscord - удаляет сообщение из Discord
func (d *BotDiscord) DeleteMessageFromDiscord(streamer *model.Streamer, channelID, msgID string) {
	d.sendWithSession(streamer, func(session *discordgo.Session) error {
		err := session.ChannelMessageDelete(channelID, msgID)
		if err != nil {
			return fmt.Errorf("ошибка удаления сообщения %s на канале %s: %v", msgID, channelID, err)
		}
		logging.Log("Discord", logrus.InfoLevel, fmt.Sprintf("Сообщение %s успешно удалено из канала %s", msgID, channelID))
		return nil
	})
}

// formatPrefix - возвращает форматированный префикс для уведомлений
func formatPrefix(prefix string) string {
	if strings.HasPrefix(prefix, "@") {
		return prefix // Прямое использование, если это @everyone или @here
	}
	return fmt.Sprintf("<@&%s>", prefix) // Использование ID роли
}
