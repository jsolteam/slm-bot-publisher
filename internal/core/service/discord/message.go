package discord

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

func FormatTelegramMessageToDiscord(message string, entities []tgbotapi.MessageEntity) string {
	if message == "" || entities == nil {
		return message
	}

	// Преобразуем строку в руны для корректной работы с символами Unicode
	runes := []rune(message)
	n := len(runes)

	// Массив для хранения открывающих и закрывающих тегов для каждой позиции
	openTags := make([]string, n+1)  // Открывающие теги
	closeTags := make([]string, n+1) // Закрывающие теги

	// Обрабатываем сущности
	for _, entity := range entities {
		entityStart := entity.Offset
		entityEnd := entity.Offset + entity.Length

		switch entity.Type {
		case "bold":
			openTags[entityStart] += "**"
			closeTags[entityEnd] = "**" + closeTags[entityEnd]
		case "italic":
			openTags[entityStart] += "*"
			closeTags[entityEnd] = "*" + closeTags[entityEnd]
		case "underline":
			openTags[entityStart] += "__"
			closeTags[entityEnd] = "__" + closeTags[entityEnd]
		case "strikethrough":
			openTags[entityStart] += "~~"
			closeTags[entityEnd] = "~~" + closeTags[entityEnd]
		case "code":
			openTags[entityStart] += "`"
			closeTags[entityEnd] = "`" + closeTags[entityEnd]
		case "pre":
			openTags[entityStart] += "```"
			closeTags[entityEnd] = "```" + closeTags[entityEnd]
		case "text_link":
			openTags[entityStart] += "["
			closeTags[entityEnd] = "](" + entity.URL + ")" + closeTags[entityEnd]
		case "text_mention":
			openTags[entityStart] += "[@" + entity.User.FirstName
			closeTags[entityEnd] = "](https://t.me/" + entity.User.UserName + ")" + closeTags[entityEnd]
		}
	}

	var formattedText strings.Builder
	for i := 0; i < n; i++ {
		formattedText.WriteString(openTags[i])
		formattedText.WriteRune(runes[i])
		formattedText.WriteString(closeTags[i+1])
	}

	return formattedText.String()
}
