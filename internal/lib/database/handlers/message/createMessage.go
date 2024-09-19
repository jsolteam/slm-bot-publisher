package message

import modeldb "slm-bot-publisher/internal/lib/database/model"

func (h *HandlerDBMessage) CreateMessage(message *modeldb.Message) error {
	return h.DB.Create(message).Error
}
