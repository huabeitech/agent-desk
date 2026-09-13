package slack

// SendMessageRequest represents payload for Slack chat.postMessage API.
type SendMessageRequest struct {
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	ParseMode string `json:"parse,omitempty"`
}

// SendMessageResponse represents response from Slack Web API.
type SendMessageResponse struct {
	OK      bool   `json:"ok"`
	Channel string `json:"channel,omitempty"`
	TS      string `json:"ts,omitempty"`
	Error   string `json:"error,omitempty"`
}

// EventCallback represents incoming Slack Events API payload.
type EventCallback struct {
	Token     string `json:"token"`
	TeamID    string `json:"team_id"`
	APIAppID  string `json:"api_app_id"`
	Type      string `json:"type"`      // url_verification | event_callback
	Challenge string `json:"challenge"` // for url_verification
	Event     *struct {
		Type        string `json:"type"` // message | app_mention
		User        string `json:"user"`
		Text        string `json:"text"`
		TS          string `json:"ts"`
		ThreadTS    string `json:"thread_ts,omitempty"`
		Channel     string `json:"channel"`
		ChannelType string `json:"channel_type"` // im | channel | group
		BotID       string `json:"bot_id,omitempty"`
		Subtype     string `json:"subtype,omitempty"`
	} `json:"event,omitempty"`
}
