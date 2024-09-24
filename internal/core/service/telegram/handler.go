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

	if streamer != nil {
		messageContent := getMessageContent(update.ChannelPost)
		attachments, attachmentsIDs := collectAttachments(update.ChannelPost, token)

		var messageModel []modeldb.Message
		messageModel = append(messageModel, buildMessageModel(update.ChannelPost.MessageID, attachmentsIDs, true))

		repostLink := buildRepostLink(update.ChannelPost.Chat.UserName, update.ChannelPost.MessageID)
		discordBot.SendMessageToDiscord(streamer, messageContent, attachments, messageModel, repostLink)
	}
}

func HandleTelegramUpdateGroup(updates []tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(updates[0].ChannelPost.Chat.ID)

	if streamer != nil {
		messageContent := getMessageContent(updates[0].ChannelPost)

		var attachments []*discordgo.File
		var messageModel []modeldb.Message
		for idx, update := range updates {
			attachmentsTG, attachmentsIDs := collectAttachments(update.ChannelPost, token)
			attachments = append(attachments, attachmentsTG...)
			messageModel = append(messageModel, buildMessageModel(update.ChannelPost.MessageID, attachmentsIDs, idx == 0))
		}

		repostLink := buildRepostLink(updates[0].ChannelPost.Chat.UserName, updates[0].ChannelPost.MessageID)
		discordBot.SendMessageToDiscord(streamer, messageContent, attachments, messageModel, repostLink)
	}
}

func HandleTelegramRepostUpdate(updates []tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(updates[0].ChannelPost.Chat.ID)
	channelPost := updates[0].ChannelPost
	channelRepostInfo := channelPost.ForwardFromChat

	if streamer != nil && channelRepostInfo != nil {
		messageContent := getMessageContent(channelPost)
		repostLink := buildRepostLink(channelRepostInfo.UserName, channelPost.ForwardFromMessageID)
		var attachments []*discordgo.File

		if messageContent == "" {
			messageContent = "-----------------------------------------"
		}

		discordRepost := model.DiscordRepost{
			ChannelName:    channelRepostInfo.Title,
			ChannelAvatar:  GetRepostChannelAvatar(channelRepostInfo.ID, token),
			MessageContent: messageContent,
			RepostLink:     repostLink,
		}

		var messageModel []modeldb.Message

		if len(updates) > 1 {
			for idx, update := range updates {
				attachmentsTG, attachmentsIDs := collectAttachments(update.ChannelPost, token)
				attachments = append(attachments, attachmentsTG...)
				messageModel = append(messageModel, buildMessageModel(update.ChannelPost.MessageID, attachmentsIDs, idx == 0))
			}
		} else {
			if channelPost.Photo == nil {
				attachmentsTG, attachmentsIDs := collectAttachments(updates[0].ChannelPost, token)
				attachments = append(attachments, attachmentsTG...)
				messageModel = append(messageModel, buildMessageModel(updates[0].ChannelPost.MessageID, attachmentsIDs, true))
			} else {
				repostPhoto := getLargestPhotoURL(channelPost, token)
				discordRepost.PhotoLink = repostPhoto
				_, attachmentsIDs := collectAttachments(updates[0].ChannelPost, token)
				messageModel = append(messageModel, buildMessageModel(updates[0].ChannelPost.MessageID, attachmentsIDs, true))
			}
		}

		discordBot.SendRepostToDiscord(streamer, discordRepost, attachments, messageModel)
	}
}

func HandleTelegramEditUpdate(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, DBHandlers *handlers.DBHandlers) {
	streamer := storage.GetStreamerByTelegramID(update.EditedChannelPost.Chat.ID)
	channelPost := update.EditedChannelPost

	messageContent := getMessageContent(channelPost)

	if streamer != nil {
		for _, channel := range streamer.DiscordChannels {
			messageIDs, err := DBHandlers.MessageHandlers.GetMessageByID(channel.ChannelID, channelPost.MessageID)
			if err != nil || len(messageIDs) == 0 {
				continue
			}

			discordBot.EditMessageOnDiscord(streamer, &channel, messageContent, messageIDs[0].DiscordMsgID)
		}
	}
}

func HandleTelegramCommand(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string, DBHandlers *handlers.DBHandlers) {
	streamer := storage.GetStreamerByTelegramID(update.ChannelPost.Chat.ID)
	currentMsgID := update.ChannelPost.MessageID

	if streamer != nil {
		command := strings.Split(update.ChannelPost.Text, " ")[0]
		commandsTelegram := map[string]CommandHandler{
			"/delete": commandTelegramDelete,
		}

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

	for idx, channel := range streamer.DiscordChannels {
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

func collectAttachments(channelPost *tgbotapi.Message, token string) ([]*discordgo.File, []string) {
	var attachments []*discordgo.File
	var attachmentIDs []string

	addAttachment := func(fileID, fileName string) {
		data := GetFileFromTelegram(fileID, token)
		if len(data) > 0 {
			attachments = append(attachments, &discordgo.File{
				Name:   fileName,
				Reader: bytes.NewReader(data),
			})
			attachmentIDs = append(attachmentIDs, fileID)
		}
	}

	processMedia(channelPost, addAttachment)

	return attachments, attachmentIDs
}

func getMessageContent(channelPost *tgbotapi.Message) string {
	text := channelPost.Text

	if text == "" {
		text = channelPost.Caption
	}
	return discord.FormatTelegramMessageToDiscord(text, channelPost.Entities)
}

func buildMessageModel(messageID int, attachmentsIDs []string, isMainPost bool) modeldb.Message {
	msg := modeldb.Message{
		MainPost:      isMainPost,
		TelegramMsgID: messageID,
	}
	if len(attachmentsIDs) > 0 {
		msg.TelegramAttachmentID = attachmentsIDs[0]
	}
	return msg
}

func buildRepostLink(username string, messageID int) string {
	if username != "" && messageID != 0 {
		return fmt.Sprintf("https://t.me/%s/%d", username, messageID)
	}
	return ""
}

func getLargestPhotoURL(channelPost *tgbotapi.Message, token string) string {
	if channelPost.Photo != nil && len(channelPost.Photo) > 0 {
		largestPhoto := channelPost.Photo[len(channelPost.Photo)-1]
		return GetFileURLFromTelegram(largestPhoto.FileID, token)
	}
	return ""
}

func processMedia(channelPost *tgbotapi.Message, addAttachment func(fileID, fileName string)) {
	if channelPost.Photo != nil && len(channelPost.Photo) > 0 {
		largestPhoto := channelPost.Photo[len(channelPost.Photo)-1]
		addAttachment(largestPhoto.FileID, "photo.jpg")
	}

	// Обрабатываем видео
	if channelPost.Video != nil {
		addAttachment(channelPost.Video.FileID, "video.mp4")
	}

	// Обрабатываем видеокружки (VideoNote)
	if channelPost.VideoNote != nil {
		addAttachment(channelPost.VideoNote.FileID, "videonote.mp4")
	}

	// Обрабатываем аудио
	if channelPost.Audio != nil {
		addAttachment(channelPost.Audio.FileID, "audio.mp3")
	}

	// Обрабатываем голосовые сообщения
	if channelPost.Voice != nil {
		addAttachment(channelPost.Voice.FileID, "voice.ogg")
	}

	// Обрабатываем документы
	if channelPost.Document != nil {
		addAttachment(channelPost.Document.FileID, channelPost.Document.FileName)
	}

	// Обрабатываем анимации (GIF) - временно не работает корректно
	//if channelPost.Animation != nil {
	//	addAttachment(channelPost.Animation.FileID, "animation.gif", token)
	//}

	// Обрабатываем стикеры
	if channelPost.Sticker != nil {
		addAttachment(channelPost.Sticker.FileID, "sticker.webp")
	}
}
