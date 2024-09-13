package storage

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"slm-bot-publisher/internal/core/model"
	"slm-bot-publisher/logging"
)

type Storage struct {
	Streamers []model.Streamer
}

func NewStorage(dataFile string) *Storage {
	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		logging.Log("Система", logrus.PanicLevel, fmt.Sprintf("Ошибка загрузки файла стримеров: %v", err))
	}

	var streamers []model.Streamer
	err = json.Unmarshal(data, &streamers)
	if err != nil {
		logging.Log("Система", logrus.PanicLevel, fmt.Sprintf("Ошибка расшифровки файла стримеров: %v", err))
	}

	return &Storage{Streamers: streamers}
}

func (s *Storage) GetStreamerByTelegramID(telegramID int64) *model.Streamer {
	for _, streamer := range s.Streamers {
		if streamer.TelegramChannelID == telegramID {
			return &streamer
		}
	}
	return nil
}
