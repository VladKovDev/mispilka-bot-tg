package main

import (
	"context"
	"log"
	"mispilkabot/config"
	"mispilkabot/internal/server"
	tgbot "mispilkabot/internal/telegram"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	tgAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to create bot API: %v", err)
	}

	tgAPI.Debug = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := server.New(cfg)

	bot := tgbot.NewBot(tgAPI, cfg)

	// Wire up the generate invite link callback
	srv.SetGenerateInviteLinkCallback(bot.GenerateInviteLink)

	// Wire up the invite message callback
	srv.SetInviteMessageCallback(func(chatID, inviteLink string) {
		bot.SendInviteMessage(chatID, inviteLink)
	})

	// Start HTTP server in a separate goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("HTTP server error: %v", err)
			serverErr <- err
		}
	}()

	// Start the bot in a separate goroutine
	botDone := make(chan struct{})
	go func() {
		defer close(botDone)
		bot.Start()
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal, initiating graceful shutdown...")
		cancel()
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		cancel()
	}

	// Wait for bot to finish (this may never happen without proper shutdown support)
	<-botDone
	log.Println("Bot stopped")
}
