package bot

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot *tgbotapi.BotAPI
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{bot: bot}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	err := services.SetSchedules(func(chatID string) {
		b.SendMessage(chatID)
	})

	if err != nil {
		fmt.Printf("SetSchedules error: %v", err)
	}

	b.handleUpdates(b.initUpdatesChanel())
}

func (b *Bot) initUpdatesChanel() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return b.bot.GetUpdatesChan(u)
}

func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.CallbackQuery != nil {
			callback := update.CallbackQuery
			b.handleCallbackQuery(callback)

		}
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
		}
	}
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	switch callback.Data {
	case "accept":
		accept(b, callback)
	case "decline":
		declaine(b, callback)
	}
}

func accept(b *Bot, callBack *tgbotapi.CallbackQuery) {
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callBack.From.ID,
		callBack.Message.MessageID,
		dataButton("✅Принято", "decline"))
	b.bot.Send(edit)
}

func declaine(b *Bot, callBack *tgbotapi.CallbackQuery) {
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callBack.From.ID,
		callBack.Message.MessageID,
		dataButton("🔳Принять", "accept"))
	b.bot.Send(edit)
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")

	switch message.Command() {
	case "start":
		b.startCommand(message)
	case "help":
		chatID := message.Chat.ID
		services.SetNextSchedule(fmt.Sprint(chatID), b.SendMessage)
	default:
		msg.Text = "I don't know that command"
		if _, err := b.bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func (b *Bot) startCommand(message *tgbotapi.Message) {
	text, err := services.GetMessage("start")
	if err != nil {
		log.Printf("message fetching error: %v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)

	msg.ReplyMarkup = dataButton("🔳Принять", "accept")

	if _, err := b.bot.Send(msg); err != nil {
		log.Panic(err)
	}

}

func (b *Bot) SendMessage(chatID string) {
	data, err := services.GetPerson(chatID)
	if err != nil {
		log.Printf("person data fetching error: %v", err)
		return
	}

	if !data.IsMessaging {
		return
	}

	last, err := services.LastMessage(data.MessagesList)
	if err != nil {
		return
	}

	text, err := services.GetMessage(last)
	if err != nil {
		log.Printf("message fetching error: %v", err)
		return
	}

	msg := tgbotapi.NewMessage(parseID(chatID), text)

	url, buttonText, err := services.GetUrlButton(last)
	if err != nil {
		return
	}
	if !(url == "" || buttonText == "") {
		keyboard := linkButton(url, buttonText)
		msg.ReplyMarkup = keyboard
	}

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("send error to %s: %v", chatID, err)
		return
	}

	data.MessagesList = data.MessagesList[:len(data.MessagesList)-1]
	services.ChangePerson(chatID, data)

	services.SetNextSchedule(chatID, b.SendMessage)
}

func linkButton(url string, buttonText string) tgbotapi.InlineKeyboardMarkup {
	urlBtn := tgbotapi.NewInlineKeyboardButtonURL(buttonText, url)
	row := tgbotapi.NewInlineKeyboardRow(urlBtn)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func dataButton(text string, calldata string) tgbotapi.InlineKeyboardMarkup {
	btn := tgbotapi.NewInlineKeyboardButtonData(text, calldata)
	row := tgbotapi.NewInlineKeyboardRow(btn)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func parseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}
