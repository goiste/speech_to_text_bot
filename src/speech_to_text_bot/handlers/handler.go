package handlers

import (
	"context"
	"fmt"

	"speech_to_text_bot/yandex"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	ycl        *yandex.YClient
	botToken   string
	ownerID    int64
	trustedIDs []int64
	bot        *tgbotapi.BotAPI
}

func New(ycl *yandex.YClient, botToken string, ownerID int64, trustedIDs []int64) *Handler {
	return &Handler{
		ycl:        ycl,
		botToken:   botToken,
		ownerID:    ownerID,
		trustedIDs: trustedIDs,
	}
}

func (h *Handler) Start(ctx context.Context, errChan chan error) {
	bot, err := tgbotapi.NewBotAPI(h.botToken)
	if err != nil {
		errChan <- fmt.Errorf("bot initialization error: %w", err)
		return
	}

	h.bot = bot

	updChan := h.bot.GetUpdatesChan(tgbotapi.UpdateConfig{})

	for {
		select {
		case upd := <-updChan:
			go h.handleUpdate(upd, errChan)
		case <-ctx.Done():
			return
		}
	}
}
