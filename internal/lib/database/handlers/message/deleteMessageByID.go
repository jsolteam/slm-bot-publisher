package message

import modeldb "slm-bot-publisher/internal/lib/database/model"

func (h *HandlerDBMessage) DeleteMessageByID(telegramMsgID uint64) error {
	return h.DB.Where("telegram_msg_id = ?", telegramMsgID).Delete(&modeldb.Message{}).Error
}
