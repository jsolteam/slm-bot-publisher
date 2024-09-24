package database

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"slm-bot-publisher/internal/lib/database/handlers"
	"slm-bot-publisher/internal/lib/database/handlers/message"
	modeldb "slm-bot-publisher/internal/lib/database/model"
	"slm-bot-publisher/logging"
	"time"
)

func InitDB(dbFilePath string) *handlers.DBHandlers {
	// Создаем файл базы данных, если он не существует
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		logging.Log("Database", logrus.InfoLevel, fmt.Sprintf("Создание базы данных по адресу: %s", dbFilePath))
		file, err := os.Create(dbFilePath)
		if err != nil {
			logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка создания файла базы данных: %v", err))
			return nil
		}
		file.Close()
	}

	// Создание файла для логов запросов базы данных
	logFile, err := os.OpenFile("logs/db_queries.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка создания файла логов: %v", err))
		return nil
	}

	// Настройка GORM для логирования запросов в файл
	newLogger := logger.New(
		log.New(logFile, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка подключения к базе данных: %v", err))
		return nil
	}

	// Автомиграция модели Message
	err = db.AutoMigrate(&modeldb.Message{})
	if err != nil {
		logging.Log("Database", logrus.PanicLevel, fmt.Sprintf("Ошибка автомиграции моделей: %v", err))
		return nil
	}

	// Инициализация хендлеров для работы с сообщениями
	messageHandler := message.NewHandlerDBMessage(db)

	return &handlers.DBHandlers{
		DB:              db,
		MessageHandlers: messageHandler,
	}
}
