package config

import (
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"slm-bot-publisher/logging"
)

type Config struct {
	TelegramToken string
	StreamerData  string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		logging.Log("Система", logrus.PanicLevel, "Ошибка загрузки .env файла")
	}

	config := &Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		StreamerData:  os.Getenv("STREAMER_DATA_FILE"),
	}

	return config
}
