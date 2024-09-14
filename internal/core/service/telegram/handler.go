package telegram

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"slm-bot-publisher/internal/core/model"
	"slm-bot-publisher/internal/core/service/discord"
	"slm-bot-publisher/internal/lib/storage"
)

func HandleTelegramUpdate(update tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(update.ChannelPost.Chat.ID)
	channelPost := update.ChannelPost

	if streamer != nil {
		messageContent := channelPost.Text
		if messageContent == "" {
			messageContent = channelPost.Caption
		}

		attachments := collectAttachments(channelPost, token)
		discordBot.SendMessageToDiscord(streamer, messageContent, attachments)
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
		for _, message := range updates {
			attachments = append(attachments, collectAttachments(message.ChannelPost, token)...)
		}

		discordBot.SendMessageToDiscord(streamer, messageContent, attachments)
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

// Функция для сбора вложений
func collectAttachments(channelPost *tgbotapi.Message, token string) []*discordgo.File {
	var attachments []*discordgo.File

	// Функция для обработки различных вложений
	addAttachment := func(fileID, fileName string, token string) {
		data := GetFileFromTelegram(fileID, token)
		if len(data) > 0 {
			attachments = append(attachments, &discordgo.File{
				Name:   fileName,
				Reader: bytes.NewReader(data),
			})
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

	return attachments
}
