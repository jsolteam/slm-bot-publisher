package main

import (
	"github.com/sirupsen/logrus"
	"slm-bot-publisher/config"
	"slm-bot-publisher/internal/core/service/discord"
	"slm-bot-publisher/internal/core/service/telegram"
	"slm-bot-publisher/internal/lib/database"
	"slm-bot-publisher/internal/lib/storage"
	"slm-bot-publisher/logging"
	"time"
)

func main() {
	logger := logging.SetupLogger()
	logger.Info("slm-bot-publisher by JSOL Team")

	configData := config.LoadConfig()
	storageData := storage.NewStorage(configData.StreamerData)

	dbHandlers := database.InitDB(configData.DatabasePath)

	discordBot := discord.NewDiscordBot(storageData, configData.TelegramToken, dbHandlers)
	telegramBot := telegram.NewTelegramBot(configData, storageData, discordBot, 10*time.Second, 3*time.Second, dbHandlers)

	telegramBot.ListenUpdates()

	logging.Log("Система", logrus.InfoLevel, "Бот приступил к работе...")
	//select {}
}
