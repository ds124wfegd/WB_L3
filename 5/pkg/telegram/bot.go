package telegram

import (
	"fmt"
	"net/http"
	"net/url"
)

type Bot struct {
	token   string
	baseURL string
}

func NewBot(token string) *Bot {
	return &Bot{
		token:   token,
		baseURL: "https://api.telegram.org/bot" + token,
	}
}

func (b *Bot) SendMessage(chatID, text string) error {
	endpoint := b.baseURL + "/sendMessage"

	params := url.Values{}
	params.Add("chat_id", chatID)
	params.Add("text", text)

	resp, err := http.PostForm(endpoint, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error: %s", resp.Status)
	}

	return nil
}
