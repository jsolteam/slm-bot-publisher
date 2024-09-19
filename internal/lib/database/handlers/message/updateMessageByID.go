package message

import modeldb "slm-bot-publisher/internal/lib/database/model"

func (h *HandlerDBMessage) UpdateMessageByID(telegramMsgID uint64, message modeldb.Message) error {
	return h.DB.Model(&modeldb.Message{}).Where("telegram_msg_id = ?", telegramMsgID).Updates(message).Error
}
