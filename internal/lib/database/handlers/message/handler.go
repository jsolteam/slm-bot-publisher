package message

import "gorm.io/gorm"

type HandlerDBMessage struct {
	DB *gorm.DB
}

func NewHandlerDBMessage(db *gorm.DB) *HandlerDBMessage {
	return &HandlerDBMessage{DB: db}
}
