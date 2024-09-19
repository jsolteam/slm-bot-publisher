package handlers

import (
	"gorm.io/gorm"
	"slm-bot-publisher/internal/lib/database/handlers/message"
)

type DBHandlers struct {
	DB              *gorm.DB
	MessageHandlers *message.HandlerDBMessage
}
