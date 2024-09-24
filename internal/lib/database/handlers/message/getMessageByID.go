package message

import modeldb "slm-bot-publisher/internal/lib/database/model"

func (h *HandlerDBMessage) GetMessageByID(channelID string, telegramMsgID int) ([]modeldb.Message, error) {
	var message modeldb.Message

	err := h.DB.Where("channel_id = ? AND telegram_msg_id = ?", channelID, telegramMsgID).First(&message).Error
	if err != nil {
		return nil, err
	}

	var relatedMessages []modeldb.Message
	err = h.DB.Where("discord_msg_id = ?", message.DiscordMsgID).Find(&relatedMessages).Error
	if err != nil {
		return nil, err
	}

	return relatedMessages, nil
}
