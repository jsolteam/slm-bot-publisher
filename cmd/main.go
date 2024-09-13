package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"slm-bot-publisher/config"
	"slm-bot-publisher/internal/core/service/discord"
	"slm-bot-publisher/internal/core/service/telegram"
	"slm-bot-publisher/internal/lib/storage"
	"slm-bot-publisher/logging"
	"time"
)

func main() {
	logger := logging.SetupLogger()
	logger.Info("slm-bot-publisher by JSOL Team")

	configData := config.LoadConfig()
	storageData := storage.NewStorage(configData.StreamerData)

	discordBot := discord.NewDiscordBot(storageData, configData.TelegramToken)
	telegramBot := telegram.NewTelegramBot(
		configData,
		func(update tgbotapi.Update) {
			telegram.HandleTelegramUpdate(update, storageData, discordBot, configData.TelegramToken)
		},
		func(updates []tgbotapi.Update) {
			telegram.HandleTelegramUpdateGroup(updates, storageData, discordBot, configData.TelegramToken)
		},
		10*time.Second,
		3*time.Second)

	telegramBot.ListenUpdates()

	logging.Log("Система", logrus.InfoLevel, "Бот приступил к работе...")
	//select {}
}
