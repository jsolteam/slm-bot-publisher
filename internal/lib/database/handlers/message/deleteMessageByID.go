package message

import modeldb "slm-bot-publisher/internal/lib/database/model"

func (h *HandlerDBMessage) DeleteMessageByID(channelID string, telegramMsgID int) error {
	var message modeldb.Message

	err := h.DB.Where("channel_id = ? AND telegram_msg_id = ?", channelID, telegramMsgID).First(&message).Error
	if err != nil {
		return err
	}

	err = h.DB.Where("discord_msg_id = ?", message.DiscordMsgID).Delete(&modeldb.Message{}).Error
	if err != nil {
		return err
	}

	return nil
}
