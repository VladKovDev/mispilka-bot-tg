package telegram

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot *tgbotapi.BotAPI
}

type Media []interface{}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{bot: bot}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	services.CheckStorage("data/users.json")
	services.CheckStorage("data/schedule_backup.json")
	services.CheckStorage("data/messages.json")

	err := services.SetSchedules(func(chatID string) {
		b.sendMessage(chatID)
	})

	if err != nil {
		fmt.Printf("SetSchedules error: %v", err)
	}

	privateChatID := parseID(os.Getenv("PRIVATE_GROUP_ID"))

	b.handleUpdates(b.initUpdatesChanel(), privateChatID)
}

func (b *Bot) initUpdatesChanel() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return b.bot.GetUpdatesChan(u)
}

func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel, privateChatID int64) {
	for update := range updates {
		chatID := update.FromChat().ID
		if chatID == privateChatID {
			if update.Message != nil && len(update.Message.NewChatMembers) > 0 {
				for _, newUser := range update.Message.NewChatMembers {
					services.ChangeIsMessaging(fmt.Sprint(newUser.ID), false)
				}
			}
			continue
		}
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
	// case "decline":
	// declaine(b, callback)
	default:
		callback := tgbotapi.NewCallback(callback.ID, "")
		b.bot.Send(callback)
		return

	}
}

func accept(b *Bot, callBack *tgbotapi.CallbackQuery) {
	services.ChangeIsMessaging(fmt.Sprint(callBack.From.ID), true)
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callBack.From.ID,
		callBack.Message.MessageID,
		dataButton("✅ Принято", "decline"))
	b.bot.Send(edit)

	services.SetSchedule(time.Now(), fmt.Sprint(callBack.From.ID), b.sendMessage)
}

func declaine(b *Bot, callBack *tgbotapi.CallbackQuery) {
	services.ChangeIsMessaging(fmt.Sprint(callBack.From.ID), false)
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callBack.From.ID,
		callBack.Message.MessageID,
		dataButton("🔲 Принимаю", "accept"))
	b.bot.Send(edit)
}


func (b *Bot) sendMessage(chatID string) {
	data, err := services.GetUser(chatID)
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

	text, err := services.GetMessageText(last)
	if err != nil {
		log.Printf("message fetching error: %v", err)
		return
	}

	url, buttonText, err := services.GetURLButton(last)
	if err != nil {
		return
	}
	var msg tgbotapi.Chattable
	photoPath, err := services.GetPhoto(last)
	if err != nil {
		m := tgbotapi.NewMessage(parseID(chatID), text)
		m.ParseMode = "HTML"
		if !(url == "" || buttonText == "") {
			keyboard := linkButton(url, buttonText)
			m.ReplyMarkup = keyboard
		}
		msg = m

	} else {
		p := tgbotapi.NewPhoto(parseID(chatID), tgbotapi.FilePath(photoPath))
		p.Caption = text
		p.ParseMode = "HTML"
		if !(url == "" || buttonText == "") {
			keyboard := linkButton(url, buttonText)
			p.ReplyMarkup = keyboard
		}
		msg = p
	}
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("send error to %s: %v", chatID, err)
		return
	}

	data.MessagesList = data.MessagesList[:len(data.MessagesList)-1]
	services.ChangeUser(chatID, data)

	last, err = services.LastMessage(data.MessagesList)
	if err != nil {
		return
	}

	services.SetNextSchedule(chatID, last, b.sendMessage)
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

func getMedia() (media Media) {
	files := []string{
		"assets/documents/ОФЕРТА.docx",
		"assets/documents/Политика конфиденциальности.docx",
		"assets/documents/Согласие ТГ БОТ.docx",
	}
	for _, f := range files {
		media = append(media, tgbotapi.NewInputMediaDocument(tgbotapi.FilePath(f)))
	}

	return media
}
