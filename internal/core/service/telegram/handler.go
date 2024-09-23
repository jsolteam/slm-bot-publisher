package telegram

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"slm-bot-publisher/internal/core/model"
	"slm-bot-publisher/internal/core/service/discord"
	"slm-bot-publisher/internal/lib/database/handlers"
	modeldb "slm-bot-publisher/internal/lib/database/model"
	"slm-bot-publisher/internal/lib/storage"
	"slm-bot-publisher/logging"
	"strings"
)

type CommandHandler func(update tgbotapi.Update, streamer *model.Streamer, bot *tgbotapi.BotAPI, discordBot *discord.BotDiscord, DBHandlers *handlers.DBHandlers)

func HandleTelegramUpdate(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(update.ChannelPost.Chat.ID)
	channelPost := update.ChannelPost

	if streamer != nil {
		messageContent := channelPost.Text
		if messageContent == "" {
			messageContent = channelPost.Caption
		}

		attachments, attachmentsIDs := collectAttachments(channelPost, token)

		var messageModel []modeldb.Message
		messageModel = append(messageModel, modeldb.Message{
			MainPost:      true,
			TelegramMsgID: channelPost.MessageID,
		})
		if len(attachmentsIDs) > 0 {
			messageModel[0].TelegramAttachmentID = attachmentsIDs[0]
		}

		discordBot.SendMessageToDiscord(streamer, messageContent, attachments, messageModel)
	}
}

func HandleTelegramUpdateGroup(updates []tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(updates[0].ChannelPost.Chat.ID)

	if streamer != nil {
		messageContent := updates[0].ChannelPost.Text
		if messageContent == "" {
			messageContent = updates[0].ChannelPost.Caption
		}

		var attachments []*discordgo.File
		var messageModel []modeldb.Message
		for idx, message := range updates {
			messageModel = append(messageModel, modeldb.Message{
				MainPost:      idx == 0,
				TelegramMsgID: message.ChannelPost.MessageID,
			})

			attachmentsTG, attachmentsIDs := collectAttachments(message.ChannelPost, token)
			attachments = append(attachments, attachmentsTG...)

			if len(attachmentsIDs) > 0 {
				messageModel[idx].TelegramAttachmentID = attachmentsIDs[0]
			}
		}

		discordBot.SendMessageToDiscord(streamer, messageContent, attachments, messageModel)
	}
}

func HandleTelegramRepostUpdate(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(update.ChannelPost.Chat.ID)
	channelPost := update.ChannelPost
	channelRepostInfo := channelPost.ForwardFromChat

	if streamer != nil && channelRepostInfo != nil {
		messageContent := channelPost.Text
		if messageContent == "" {
			messageContent = channelPost.Caption
		}

		repostPhoto := ""
		if channelPost.Photo != nil && channelPost.MediaGroupID == "" {
			largestPhoto := channelPost.Photo[len(channelPost.Photo)-1]
			repostPhoto = GetFileURLFromTelegram(largestPhoto.FileID, token)
		}

		if messageContent == "" && repostPhoto == "" {
			return
		}

		repostChannelName := channelRepostInfo.Title
		repostChannelAvatar := GetRepostChannelAvatar(channelRepostInfo.ID, token)

		var repostLink string
		if channelRepostInfo.UserName != "" && channelPost.ForwardFromMessageID != 0 {
			repostLink = fmt.Sprintf("https://t.me/%s/%d", channelRepostInfo.UserName, channelPost.ForwardFromMessageID)
		}

		var discordRepost = model.DiscordRepost{
			ChannelName:    repostChannelName,
			ChannelAvatar:  repostChannelAvatar,
			MessageContent: messageContent,
			PhotoLink:      repostPhoto,
			RepostLink:     repostLink,
		}

		discordBot.SendRepostToDiscord(streamer, discordRepost)
	}
}

func HandleTelegramEditUpdate(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(update.ChannelPost.Chat.ID)
	//channelPost := update.EditedChannelPost

	if streamer != nil {

	}
}

func HandleTelegramCommand(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string, DBHandlers *handlers.DBHandlers) {
	streamer := storage.GetStreamerByTelegramID(update.ChannelPost.Chat.ID)
	currentMsgID := update.ChannelPost.MessageID
	commandsTelegram := map[string]CommandHandler{
		"/delete": commandTelegramDelete,
	}

	if streamer != nil {
		command := strings.Split(update.ChannelPost.Text, " ")[0]
		if handler, exists := commandsTelegram[command]; exists {
			bot, err := tgbotapi.NewBotAPI(token)
			if err != nil {
				logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка при подключении к боту: %v", err))
				return
			}
			DeletePostFromChannel(update.ChannelPost.Chat.ID, currentMsgID, bot)
			handler(update, streamer, bot, discordBot, DBHandlers)
		} else {
			logging.Log("Telegram", logrus.InfoLevel, fmt.Sprintf("Неизвестная команда: %s", command))
		}
	}
}

func commandTelegramDelete(update tgbotapi.Update, streamer *model.Streamer, bot *tgbotapi.BotAPI, discordBot *discord.BotDiscord, DBHandlers *handlers.DBHandlers) {
	if update.ChannelPost.ReplyToMessage == nil {
		return
	}

	deleteMsgID := update.ChannelPost.ReplyToMessage.MessageID
	channelIDs := streamer.DiscordChannels

	for idx, channel := range channelIDs {
		messageIDs, err := DBHandlers.MessageHandlers.GetMessageByID(channel.ChannelID, deleteMsgID)
		if err != nil || len(messageIDs) == 0 {
			continue
		}

		if idx == 0 {
			for _, msg := range messageIDs {
				DeletePostFromChannel(update.ChannelPost.Chat.ID, msg.TelegramMsgID, bot)
			}
		}

		discordBot.DeleteMessageFromDiscord(streamer, channel.ChannelID, messageIDs[0].DiscordMsgID)
		err = DBHandlers.MessageHandlers.DeleteMessageByID(channel.ChannelID, deleteMsgID)
		if err != nil {
			logging.Log("Database", logrus.ErrorLevel, fmt.Sprintf("Не удалось удалить сообщения с ID Discord %s", messageIDs[0].DiscordMsgID))
		}
	}
}

// Функция для сбора вложений
func collectAttachments(channelPost *tgbotapi.Message, token string) ([]*discordgo.File, []string) {
	var attachments []*discordgo.File
	var attachmentsIDs []string

	// Функция для обработки различных вложений
	addAttachment := func(fileID, fileName string, token string) {
		data := GetFileFromTelegram(fileID, token)
		if len(data) > 0 {
			attachments = append(attachments, &discordgo.File{
				Name:   fileName,
				Reader: bytes.NewReader(data),
			})
			attachmentsIDs = append(attachmentsIDs, fileID)
		}
	}

	// Обрабатываем фотографии
	if channelPost.Photo != nil && len(channelPost.Photo) > 0 {
		// Ищем фото с наибольшим размером
		largestPhoto := channelPost.Photo[0]
		for _, photo := range channelPost.Photo {
			if photo.FileSize > largestPhoto.FileSize {
				largestPhoto = photo
			}
		}
		addAttachment(largestPhoto.FileID, "photo.jpg", token)
	}

	// Обрабатываем видео
	if channelPost.Video != nil {
		addAttachment(channelPost.Video.FileID, "video.mp4", token)
	}

	// Обрабатываем видеокружки (VideoNote)
	if channelPost.VideoNote != nil {
		addAttachment(channelPost.VideoNote.FileID, "videonote.mp4", token)
	}

	// Обрабатываем аудио
	if channelPost.Audio != nil {
		addAttachment(channelPost.Audio.FileID, "audio.mp3", token)
	}

	// Обрабатываем голосовые сообщения
	if channelPost.Voice != nil {
		addAttachment(channelPost.Voice.FileID, "voice.ogg", token)
	}

	// Обрабатываем документы
	if channelPost.Document != nil {
		addAttachment(channelPost.Document.FileID, channelPost.Document.FileName, token)
	}

	// Обрабатываем анимации (GIF) - временно не работает корректно
	//if channelPost.Animation != nil {
	//	addAttachment(channelPost.Animation.FileID, "animation.gif", token)
	//}

	// Обрабатываем стикеры
	if channelPost.Sticker != nil {
		addAttachment(channelPost.Sticker.FileID, "sticker.webp", token)
	}

	return attachments, attachmentsIDs
}
