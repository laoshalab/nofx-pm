package telegram

import "sync"

var (
	notifyMu    sync.RWMutex
	boundChatID int64
	sendMarkdown func(chatID int64, text string)
)

// RegisterMessenger binds the active bot sender and chat ID for proactive alerts.
func RegisterMessenger(chatID int64, send func(int64, string)) {
	notifyMu.Lock()
	defer notifyMu.Unlock()
	boundChatID = chatID
	sendMarkdown = send
}

// UpdateBoundChat updates the chat ID after /start re-bind.
func UpdateBoundChat(chatID int64) {
	notifyMu.Lock()
	defer notifyMu.Unlock()
	boundChatID = chatID
}

// SendMarkdown delivers a proactive message to the bound Telegram chat.
func SendMarkdown(text string) {
	notifyMu.RLock()
	id := boundChatID
	fn := sendMarkdown
	notifyMu.RUnlock()
	if fn == nil || id == 0 || text == "" {
		return
	}
	fn(id, text)
}

// BoundChatID returns the currently bound chat (0 if unbound).
func BoundChatID() int64 {
	notifyMu.RLock()
	defer notifyMu.RUnlock()
	return boundChatID
}
