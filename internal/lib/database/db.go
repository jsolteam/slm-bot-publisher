package database

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"os"
	"slm-bot-publisher/internal/lib/database/handlers"
	"slm-bot-publisher/internal/lib/database/handlers/message"
	modeldb "slm-bot-publisher/internal/lib/database/model"
	"slm-bot-publisher/logging"
)

func InitDB(dbFilePath string) *handlers.DBHandlers {
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		logging.Log("Database", logrus.InfoLevel, fmt.Sprintf("Создание базы данных по адресу: %s", dbFilePath))
		file, err := os.Create(dbFilePath)
		if err != nil {
			logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка создания файла базы данных: %v", err))
			return nil
		}
		file.Close()
	}

	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка подключения к базе данных: %v", err))
		return nil
	}

	err = db.AutoMigrate(&modeldb.Message{})
	if err != nil {
		logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка автомиграции моделей: %v", err))
		return nil
	}

	messageHandler := message.NewHandlerDBMessage(db)

	return &handlers.DBHandlers{
		DB:              db,
		MessageHandlers: messageHandler,
	}
}
