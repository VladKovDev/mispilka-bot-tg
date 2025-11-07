package bot

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"
	"strconv"
	"strings"
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

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")
	lowerCommand := strings.ToLower(message.Command())
	switch lowerCommand {
	case "start":
		b.startCommand(message)
	case "restart":
		if err := services.AddUser(message); err != nil {
			log.Printf("Failed to add user: %v", err)
		}
	default:
		msg.Text = "I don't know that command"
		if _, err := b.bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func (b *Bot) startCommand(message *tgbotapi.Message) {
	if services.IsNewUser(fmt.Sprint(message.Chat.ID)) {
		err := services.AddUser(message)
		if err != nil {
			return
		}
	}

	text, err := services.GetMessageText("start")
	if err != nil {
		log.Printf("message fetching error: %v", err)
		return
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	// media := getMedia()
	// group := tgbotapi.MediaGroupConfig{
	// 	ChatID: message.Chat.ID,
	// 	Media:  media,
	// }

	userData, err := services.GetUser(fmt.Sprint(message.Chat.ID))
	if err == nil {
		if userData.IsMessaging {
			msg.ReplyMarkup = dataButton("✅ Принято", "decline")
		} else {
			msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")
		}
	} else {
		msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")
	}

	// _, err = b.bot.SendMediaGroup(group)
	// if err != nil {
	// 	log.Printf("send mediagroup error: %s", err)
	// 	return
	// }
	msg.ParseMode = "HTML"
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("message send error: %s", err)
		return
	}

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
