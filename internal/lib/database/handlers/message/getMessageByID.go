package message

import modeldb "slm-bot-publisher/internal/lib/database/model"

func (h *HandlerDBMessage) GetMessageByID(telegramMsgID uint64) ([]modeldb.Message, error) {
	var messages []modeldb.Message
	err := h.DB.Where("telegram_msg_id = ?", telegramMsgID).Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}
