package telegram

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"slm-bot-publisher/config"
	"slm-bot-publisher/logging"
	"sync"
	"time"
)

type UpdateGroup struct {
	Updates   []tgbotapi.Update
	Timestamp time.Time
}

type BotTelegram struct {
	Bot                  *tgbotapi.BotAPI
	updateGroups         map[string]*UpdateGroup
	updateGroupMutex     sync.Mutex
	updateHandler        func(update tgbotapi.Update)
	updateRepostHandler  func(update tgbotapi.Update)
	updateGroupHandler   func(updates []tgbotapi.Update)
	flushInterval        time.Duration
	updateGroupFlushTime time.Duration
}

func NewTelegramBot(config *config.Config, updateHandler, updateRepostHandler func(update tgbotapi.Update), updateGroupHandler func(updates []tgbotapi.Update), flushInterval, updateGroupFlushTime time.Duration) *BotTelegram {
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		logging.Log("Telegram", logrus.PanicLevel, fmt.Sprintf("%v", err))
	}
	logging.Log("Telegram", logrus.InfoLevel, "Успешное подключение к боту Telegram")

	bt := &BotTelegram{
		Bot:                  bot,
		updateGroups:         make(map[string]*UpdateGroup),
		updateHandler:        updateHandler,
		updateRepostHandler:  updateRepostHandler,
		updateGroupHandler:   updateGroupHandler,
		flushInterval:        flushInterval,
		updateGroupFlushTime: updateGroupFlushTime,
	}

	go bt.startFlushRoutine()

	return bt
}

func (t *BotTelegram) startFlushRoutine() {
	for {
		time.Sleep(t.flushInterval)
		t.flushQueue()
	}
}

func (t *BotTelegram) flushQueue() {
	t.updateGroupMutex.Lock()
	defer t.updateGroupMutex.Unlock()

	now := time.Now()

	for id, group := range t.updateGroups {
		if now.Sub(group.Timestamp) >= t.updateGroupFlushTime {
			logging.Log("Telegram", logrus.InfoLevel, fmt.Sprintf("Получено новое сообщение с канала %s", group.Updates[0].ChannelPost.Chat.Title))
			t.updateGroupHandler(group.Updates)
			delete(t.updateGroups, id)
		}
	}
}

func (t *BotTelegram) appendQueue(update tgbotapi.Update) {
	t.updateGroupMutex.Lock()
	defer t.updateGroupMutex.Unlock()

	if _, exist := t.updateGroups[update.ChannelPost.MediaGroupID]; !exist {
		t.updateGroups[update.ChannelPost.MediaGroupID] = &UpdateGroup{
			Updates:   []tgbotapi.Update{update},
			Timestamp: time.Now(),
		}
	} else {
		t.updateGroups[update.ChannelPost.MediaGroupID].Updates = append(t.updateGroups[update.ChannelPost.MediaGroupID].Updates, update)
		t.updateGroups[update.ChannelPost.MediaGroupID].Timestamp = time.Now()
	}
}

func (t *BotTelegram) ListenUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	logging.Log("Telegram", logrus.InfoLevel, "Начинается прослушка сообщений...")

	updates := t.Bot.GetUpdatesChan(u)
	for update := range updates {
		if update.ChannelPost != nil {
			if update.ChannelPost.ForwardFrom == nil && update.ChannelPost.ForwardFromChat == nil {
				if update.ChannelPost.MediaGroupID != "" {
					t.appendQueue(update)
				} else {
					logging.Log("Telegram", logrus.InfoLevel, fmt.Sprintf("Получено новое сообщение с канала %s", update.ChannelPost.Chat.Title))
					t.updateHandler(update)
				}
			} else if update.ChannelPost.ForwardFromChat != nil {
				logging.Log("Telegram", logrus.InfoLevel, fmt.Sprintf("Получено новое сообщение с канала %s", update.ChannelPost.Chat.Title))
				t.updateRepostHandler(update)
			}
		}
	}
}

func GetFileFromTelegram(fileID string, token string) []byte {
	filePathURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", token, fileID)
	resp, err := http.Get(filePathURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Не удалось получить путь к файлу: статус %d", resp.StatusCode))
		return nil
	}

	var fileData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка декодирования файла: %v", err))
		return nil
	}

	filePath, ok := fileData["result"].(map[string]interface{})["file_path"].(string)
	if !ok {
		logging.Log("Telegram", logrus.ErrorLevel, "не удалось получить путь к файлу из ответа")
		return nil
	}

	photoURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, filePath)
	photoResp, err := http.Get(photoURL)
	if err != nil {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка отправки запроса: %v", err))
		return nil
	}
	defer photoResp.Body.Close()

	if photoResp.StatusCode != http.StatusOK {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Не удалось загрузить фото: статус %d", photoResp.StatusCode))
		return nil
	}

	dataPhoto, _ := ioutil.ReadAll(photoResp.Body)

	return dataPhoto
}

func GetFileURLFromTelegram(fileID string, token string) string {
	filePathURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", token, fileID)
	resp, err := http.Get(filePathURL)
	if err != nil {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка получения информации о файле: %v", err))
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Не удалось получить путь к файлу: статус %d", resp.StatusCode))
		return ""
	}

	var fileData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка декодирования ответа: %v", err))
		return ""
	}

	filePath, ok := fileData["result"].(map[string]interface{})["file_path"].(string)
	if !ok {
		logging.Log("Telegram", logrus.ErrorLevel, "Не удалось получить путь к файлу из ответа")
		return ""
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, filePath)
	return fileURL
}

func GetRepostChannelAvatar(chatID int64, token string) string {
	chatInfoURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%d", token, chatID)

	resp, err := http.Get(chatInfoURL)
	if err != nil {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка при запросе информации о канале: %v", err))
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Не удалось получить информацию о канале: статус %d", resp.StatusCode))
		return ""
	}

	var chatData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&chatData); err != nil {
		logging.Log("Telegram", logrus.ErrorLevel, fmt.Sprintf("Ошибка декодирования ответа: %v", err))
		return ""
	}

	chatInfo, ok := chatData["result"].(map[string]interface{})
	if !ok || chatInfo["photo"] == nil {
		logging.Log("Telegram", logrus.InfoLevel, "Аватар канала не найден")
		return ""
	}

	photoInfo := chatInfo["photo"].(map[string]interface{})
	largestAvatarFileID, ok := photoInfo["big_file_id"].(string)
	if !ok {
		logging.Log("Telegram", logrus.InfoLevel, "Не удалось получить большой аватар канала")
		return ""
	}

	return GetFileURLFromTelegram(largestAvatarFileID, token)
}
