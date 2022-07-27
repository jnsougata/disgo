package interaction

import (
	"encoding/json"
	"fmt"
	"github.com/jnsougata/disgo/core/command"
	"github.com/jnsougata/disgo/core/component"
	"github.com/jnsougata/disgo/core/embed"
	"github.com/jnsougata/disgo/core/file"
	"github.com/jnsougata/disgo/core/modal"
	"github.com/jnsougata/disgo/core/router"
	"github.com/jnsougata/disgo/core/user"
)

type Message struct {
	Content         string
	Embeds          []embed.Embed
	AllowedMentions []string
	TTS             bool
	Ephemeral       bool
	SuppressEmbeds  bool
	View            component.View
	Files           []file.File
}

func (m *Message) ToBody() map[string]interface{} {
	flag := 0
	body := map[string]interface{}{}
	if m.Content != "" {
		body["content"] = m.Content
	}
	if len(m.Embeds) > 0 && len(m.Embeds) <= 25 {
		body["embeds"] = m.Embeds
	}
	if len(m.AllowedMentions) > 0 && len(m.AllowedMentions) <= 100 {
		body["allowed_mentions"] = m.AllowedMentions
	}
	if m.TTS {
		body["tts"] = true
	}
	if m.Ephemeral {
		flag |= 1 << 6
	}
	if m.SuppressEmbeds {
		flag |= 1 << 2
	}
	if m.Ephemeral || m.SuppressEmbeds {
		body["flags"] = flag
	}
	if len(m.View.ActionRows) > 0 {
		body["components"] = m.View.ToComponent()
	}
	if len(m.Files) > 0 {
		body["attachments"] = []map[string]interface{}{}
		for i, file := range m.Files {
			if len(file.Content) > 0 {
				a := map[string]interface{}{
					"id":          i,
					"filename":    file.Name,
					"description": file.Description,
				}
				body["attachments"] = append(body["attachments"].([]map[string]interface{}), a)
			}
		}
	}
	return body
}

type Option struct {
	Name    string      `json:"name"`
	Type    int         `json:"type"`
	Value   interface{} `json:"value"`
	Options []Option    `json:"options"`
	Focused bool        `json:"focused"`
}

type Data struct {
	Id       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     int                    `json:"type"`
	Resolved map[string]interface{} `json:"resolved"`
	Options  []Option               `json:"options"`
	GuildID  string                 `json:"guild_id"`
	TargetId string                 `json:"target_id"`
}

type Interaction struct {
	ID             string      `json:"id"`
	ApplicationID  string      `json:"application_id"`
	Type           int         `json:"type"`
	Data           Data        `json:"data"`
	GuildID        string      `json:"guild_id"`
	ChannelID      string      `json:"channel_id"`
	Member         interface{} `json:"member"`
	User           user.User   `json:"user"`
	Token          string      `json:"token"`
	Version        int         `json:"version"`
	Message        interface{} `json:"message"`
	AppPermissions string      `json:"app_permissions"`
	Locale         string      `json:"locale"`
	GuildLocale    string      `json:"guild_locale"`
}

func FromData(payload interface{}) *Interaction {
	i := &Interaction{}
	data, _ := json.Marshal(payload)
	_ = json.Unmarshal(data, i)
	return i
}

func (i *Interaction) SendResponse(message Message) {
	path := fmt.Sprintf("/interactions/%s/%s/callback", i.ID, i.Token)
	body := map[string]interface{}{
		"type": 4,
		"data": message.ToBody(),
	}
	r := router.New("POST", path, body, "", message.Files)
	go r.Request()
}

func (i *Interaction) Ack() {
	path := fmt.Sprintf("/interactions/%s/%s/callback", i.ID, i.Token)
	r := router.New("POST", path, map[string]interface{}{"type": 1}, "", nil)
	go r.Request()
}

func (i *Interaction) Defer(ephemeral bool) {
	payload := map[string]interface{}{"type": 5}
	if ephemeral {
		payload["data"] = map[string]interface{}{"flags": 1 << 6}
	}
	path := fmt.Sprintf("/interactions/%s/%s/callback", i.ID, i.Token)
	r := router.New("POST", path, payload, "", nil)
	go r.Request()
}

func (i *Interaction) SendModal(modal modal.Modal) {
	path := fmt.Sprintf("/interactions/%s/%s/callback", i.ID, i.Token)
	r := router.New("POST", path, modal.ToBody(), "", nil)
	go r.Request()
}

func (i *Interaction) SendAutoComplete(choices ...command.Choice) {
	payload := map[string]interface{}{
		"type": 8,
		"data": map[string]interface{}{"choices": choices},
	}
	path := fmt.Sprintf("/interactions/%s/%s/callback", i.ID, i.Token)
	r := router.New("POST", path, payload, "", nil)
	go r.Request()
}

func (i *Interaction) SendFollowup(message Message) {
	path := fmt.Sprintf("/webhooks/%s/%s", i.ApplicationID, i.Token)
	r := router.New("POST", path, message.ToBody(), "", message.Files)
	go r.Request()
}