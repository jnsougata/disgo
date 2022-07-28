package component

import (
	"github.com/jnsougata/disgo/core/embed"
	"github.com/jnsougata/disgo/core/emoji"
	"github.com/jnsougata/disgo/core/file"
	"github.com/jnsougata/disgo/core/user"
	"github.com/jnsougata/disgo/core/utils"
	"log"
)

const (
	BlueButton  = 1
	GreyButton  = 2
	GreenButton = 3
	RedButton   = 4
	LinkButton  = 5
)

var CallbackTasks = map[string]interface{}{}
var TimeoutTasks = map[string][]interface{}{}

type Message struct {
	Content         string
	Embed           embed.Embed
	Embeds          []embed.Embed
	AllowedMentions []string
	TTS             bool
	Ephemeral       bool
	SuppressEmbeds  bool
	View            View
	File            file.File
	Files           []file.File
}

func (m *Message) ToBody() map[string]interface{} {
	flag := 0
	body := map[string]interface{}{}
	if m.Content != "" {
		body["content"] = m.Content
	}
	if utils.CheckTrueEmbed(m.Embed) {
		m.Embeds = append([]embed.Embed{m.Embed}, m.Embeds...)
	}
	for i, em := range m.Embeds {
		if !utils.CheckTrueEmbed(em) {
			m.Embeds = append(m.Embeds[:i], m.Embeds[i+1:]...)
		}
	}
	if len(m.Embeds) > 10 {
		m.Embeds = m.Embeds[:10]
	}
	body["embeds"] = m.Embeds
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
	if utils.CheckTrueFile(m.File) {
		m.Files = append([]file.File{m.File}, m.Files...)
	}
	body["attachments"] = []map[string]interface{}{}
	for i, f := range m.Files {
		if utils.CheckTrueFile(f) {
			a := map[string]interface{}{
				"id":          i,
				"filename":    f.Name,
				"description": f.Description,
			}
			body["attachments"] = append(body["attachments"].([]map[string]interface{}), a)
		} else {
			m.Files = append(m.Files[:i], m.Files[i+1:]...)
		}
	}
	return body
}

type Button struct {
	Style    int
	Label    string
	Emoji    emoji.Partial
	URL      string
	Disabled bool
	CustomId string
	OnClick  func(bot user.User, ctx Context)
}

func (b *Button) ToComponent() map[string]interface{} {
	b.CustomId = utils.AssignId("")
	if b.OnClick != nil {
		CallbackTasks[b.CustomId] = b.OnClick
	}
	btn := map[string]interface{}{
		"type":      2,
		"custom_id": b.CustomId,
	}
	if b.Style != 0 {
		btn["style"] = b.Style
	} else {
		btn["style"] = BlueButton
	}
	if b.Label != "" {
		btn["label"] = b.Label
	} else {
		btn["label"] = "Button"
	}
	if b.Emoji.Id != "" {
		btn["emoji"] = b.Emoji
	}
	if b.URL != "" && b.Style == LinkButton {
		btn["url"] = b.URL
	}
	if b.Disabled {
		btn["disabled"] = true
	}
	return btn
}

type SelectOption struct {
	Label       string        `json:"label"`
	Value       string        `json:"value"`
	Description string        `json:"description,omitempty"`
	Emoji       emoji.Partial `json:"emoji,omitempty"`
	Default     bool          `json:"default,omitempty"`
}

type SelectMenu struct {
	Type        int
	CustomId    string
	Options     []SelectOption
	Placeholder string
	MinValues   int
	MaxValues   int
	Disabled    bool
	OnSelection func(bot user.User, ctx Context, values ...string)
}

func (s *SelectMenu) ToComponent() map[string]interface{} {
	s.CustomId = utils.AssignId("")
	if s.OnSelection != nil {
		CallbackTasks[s.CustomId] = s.OnSelection
	}
	menu := map[string]interface{}{"type": 3, "custom_id": s.CustomId}
	if s.Placeholder != "" {
		menu["placeholder"] = s.Placeholder
	}
	if s.MinValues >= 0 {
		menu["min_values"] = s.MinValues
	} else {
		log.Println("MinValues must be greater than or equal to 0")
	}
	if s.MaxValues <= 25 {
		menu["max_values"] = s.MaxValues
	} else {
		log.Println("MaxValues must be less than or equal to 25")
	}
	if s.Disabled {
		menu["disabled"] = true
	}
	if len(s.Options) > 0 {
		menu["options"] = s.Options
	}
	return menu
}

type ActionRow struct {
	Buttons    []Button
	SelectMenu SelectMenu
}

type View struct {
	Timeout    float64
	ActionRows []ActionRow
	OnTimeout  func(bot user.User, interaction Context)
}

func (v *View) ToComponent() []interface{} {
	if v.Timeout == 0 || v.Timeout > 14.8*60 {
		v.Timeout = 14.8 * 60
	}
	var undo = map[string]bool{}
	var c []interface{}
	if len(v.ActionRows) > 0 && len(v.ActionRows) <= 5 {
		for _, row := range v.ActionRows {
			num := 0
			tmp := map[string]interface{}{
				"type":       1,
				"components": []interface{}{},
			}
			for _, button := range row.Buttons {
				if num < 5 {
					undo[button.CustomId] = true
					if v.OnTimeout != nil {
						TimeoutTasks[button.CustomId] = []interface{}{v.Timeout, v.OnTimeout}
					}
					tmp["components"] = append(tmp["components"].([]interface{}), button.ToComponent())
					num++
				}
			}
			if len(row.SelectMenu.Options) > 0 {
				if num == 0 {
					undo[row.SelectMenu.CustomId] = true
					if v.OnTimeout != nil {
						TimeoutTasks[row.SelectMenu.CustomId] = []interface{}{v.Timeout, v.OnTimeout}
					}
					tmp["components"] = append(tmp["components"].([]interface{}), row.SelectMenu.ToComponent())
				} else {
					log.Println("Single ActionRow can contain either 1x SelectMenu or max 5x Buttons")
				}
			}
			if len(undo) > 0 {
				c = append(c, tmp)
				go utils.ScheduleDeletion(v.Timeout, CallbackTasks, undo)
			}
		}
	}
	return c
}
