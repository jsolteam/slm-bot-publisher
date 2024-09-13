# SLM Bot Publisher


<img align="right" src="https://i.ibb.co/fHHNWjL/jsol-team-white.png" height="30px" alt="JSOL Team Logo"/>

![Golang Badge](https://img.shields.io/badge/Go-1.23-blue)
![GitHub License](https://img.shields.io/github/license/jsolteam/slm-bot-publisher)
![GitHub Release](https://img.shields.io/github/v/release/jsolteam/slm-bot-publisher)


<div align="center">
  <img src="https://i.ibb.co/bvTNdyq/slm-bot-publisher-logo-with-text.png" alt="Project Logo" height="200"/>
   <p align="center">Бот для пересылки сообщений с Telegram канала в Discord канал</p>
</div>

<p align="center">
  <a href="https://t.me/squadslm">SLM Squad Telegram</a> •
  <a href="https://t.me/jsol_team">JSOL Team Telegram</a> •
  <a href="https://github.com/jsolteam">JSOL Team Github</a> •
</p>

<br/>

### Запуск

1. Установите Go, следуя официальной инструкции: [https://go.dev/doc/install](https://go.dev/doc/install)
2. Убедитесь, что Go установлен корректно:
   ```sh
   go version
   ```
3. Клонируйте репозиторий
   ```sh
   git clone https://github.com/jsolteam/slm-bot-publisher.git
   ```
4. Перейдите в папку проекта
   ```sh
   cd slm-bot-publisher
   ```
5. Установите зависимости из файла go.mod
   ```sh
   go mod tidy
   ```
6. Создайте .env файл, файл config, согласно информации ниже
7. Запустите проект
    ```sh
   go run cmd/main.go
   ```

### ENV файл

    
```shell
TELEGRAM_TOKEN=*Ваш токен Telegram бота*
STREAMER_DATA_FILE=*Располежение файла конфига .json*
```

### Конфиг


```json
[
  {
    "Name": "Test", - Имя отдельного бота Discord
    "TelegramChannelID": -44353456346, - ID Telegram канала  
    "DiscordBotToken": "HFge4rtfdb5btb", - Токен Discord бота
    "DiscordChannels": [ - Список каналов Discord
      {
        "ChannelID": "35464365365", - ID Discord канала
        "Prefix": "@everyone" - Префикс, используемый в начале сообщения
      },
      ...
    ]
  },
  ...
]

```

### Релизы

Все доступные релизы можно найти в разделе [Releases](https://github.com/jsolteam/slm-bot-publisher/releases).

### Контрибуторы

<a href="https://github.com/jsolteam/slm-bot-publisher/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=jsolteam/slm-bot-publisher" alt="contrib.rocks image" />
</a>

### Лицензия

Проект представлен MIT лицензией. Смотрите `LICENSE` для подробной информации.
