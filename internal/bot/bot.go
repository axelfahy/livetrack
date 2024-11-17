package bot

import (
	"errors"
	"log/slog"

	"github.com/mymmrac/telego"
)

type Bot struct {
	client   *telego.Bot
	channel  string
	messages []*telego.Message
	logger   *slog.Logger
	metrics  metrics
}

type metrics interface {
	MessageSent()
	MessageRemoved()
}

func New(channel, token string, logger *slog.Logger, metrics metrics) (*Bot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, err
	}
	// Retrieve information on the channel.
	botUser, err := bot.GetMe()
	if err != nil {
		return nil, err
	}

	logger.Debug("Bot", "user", botUser)

	return &Bot{
		client:   bot,
		channel:  channel,
		messages: []*telego.Message{},
		logger:   logger,
		metrics:  metrics,
	}, nil
}

func (bot *Bot) SendMessage(text string) error {
	message, err := bot.client.SendMessage(&telego.SendMessageParams{
		ChatID:    telego.ChatID{Username: bot.channel},
		Text:      text,
		ParseMode: telego.ModeMarkdown,
	})
	if err != nil {
		return err
	}

	bot.messages = append(bot.messages, message)
	bot.logger.Debug("Sent", "message", message)
	bot.metrics.MessageSent()

	return nil
}

func (bot *Bot) DeleteMessages() error {
	errs := []error{}

	for _, message := range bot.messages {
		bot.logger.Debug("Removing", "message", message)

		err := bot.client.DeleteMessage(&telego.DeleteMessageParams{
			ChatID:    telego.ChatID{Username: bot.channel},
			MessageID: message.MessageID,
		})
		if err != nil {
			bot.logger.Error("Error removing", "message", message, "error", err)
			errs = append(errs, err)
		}

		bot.metrics.MessageRemoved()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
