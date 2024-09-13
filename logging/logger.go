package logging

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

var log = logrus.New()

type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := entry.Level.String()

	module, ok := entry.Data["module"].(string)
	var logMessage string
	if ok && module != "" {
		logMessage = fmt.Sprintf("%s (%s) [%s]: %s\n", timestamp, level, module, entry.Message)
	} else {
		logMessage = fmt.Sprintf("%s (%s) [Система]: %s\n", timestamp, level, entry.Message)
	}

	return []byte(logMessage), nil
}

func SetupLogger() *logrus.Logger {
	log.SetFormatter(&CustomFormatter{})

	// Создаем директорию для логов, если она не существует
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err = os.Mkdir("logs", 0755)
		if err != nil {
			log.Fatalf("Невозможно создать директорию для логов: %v", err)
		}
	}

	// Устанавливаем файл для логов на каждый день
	fileName := fmt.Sprintf("logs/log-%s.log", time.Now().Format("2006-01-02"))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Ошибка при открытии файла логов: %v", err)
	}

	log.SetOutput(io.MultiWriter(file, os.Stdout))
	log.SetLevel(logrus.InfoLevel)

	return log
}

func Log(module string, level logrus.Level, message string) {
	entry := log.WithFields(logrus.Fields{
		"module": module,
	})

	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	case logrus.PanicLevel:
		entry.Panic(message)
	default:
		entry.Info(message)
	}
}
