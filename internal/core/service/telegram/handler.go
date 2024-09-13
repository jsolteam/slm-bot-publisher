package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

		// Получаем наибольшую фотографию
		var largestPhoto tgbotapi.PhotoSize
		if channelPost.Photo != nil {
			for _, photo := range channelPost.Photo {
				if largestPhoto.FileID == "" || photo.FileSize > largestPhoto.FileSize {
					largestPhoto = photo
				}
			}
		}

		photoData := make([][]byte, 1)
		if largestPhoto.FileID != "" {
			photoData[0] = GetPhotoFromTelegram(largestPhoto.FileID, token)
		}
		discordBot.SendMessageToDiscord(streamer, messageContent, photoData)
	}
}

func HandleTelegramUpdateGroup(updates []tgbotapi.Update, storage *storage.Storage, discordBot *discord.BotDiscord, token string) {
	streamer := storage.GetStreamerByTelegramID(updates[0].ChannelPost.Chat.ID)

	if streamer != nil {
		messageContent := updates[0].ChannelPost.Text
		if messageContent == "" {
			messageContent = updates[0].ChannelPost.Caption
		}

		var photos [][]byte
		for _, message := range updates {
			channelPost := message.ChannelPost

			// Получаем наибольшую фотографию
			var largestPhoto tgbotapi.PhotoSize
			if channelPost.Photo != nil {
				for _, photo := range channelPost.Photo {
					if largestPhoto.FileID == "" || photo.FileSize > largestPhoto.FileSize {
						largestPhoto = photo
					}
				}

				var photoData []byte
				if largestPhoto.FileID != "" {
					photoData = GetPhotoFromTelegram(largestPhoto.FileID, token)
				}
				if len(photoData) > 0 {
					photos = append(photos, photoData)
				}
			}
		}

		discordBot.SendMessageToDiscord(streamer, messageContent, photos)
	}
}
