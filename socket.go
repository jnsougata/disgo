package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/jnsougata/disgo/bot"
	"io"
	"log"
	"net/http"
	"time"
)

type runtime struct {
	SessionId   string `json:"session_id"`
	Sequence    int
	Application struct {
		Id    string  `json:"id"`
		Flags float64 `json:"flags"`
	} `json:"application"`
	User struct {
		Bot           bool    `json:"bot"`
		Avatar        string  `json:"avatar"`
		Discriminator string  `json:"discriminator"`
		Flags         float64 `json:"flags"`
		MFAEnabled    bool    `json:"mfa_enabled"`
		Username      string  `json:"username"`
		Verified      bool    `json:"verified"`
	}
}

var b *bot.User
var latency int64
var beatSent int64
var isready = false
var interval float64
var beatReceived int64
var queue []ApplicationCommand
var eventHooks = map[string]interface{}{}
var commandHooks = map[string]interface{}{}
var guilds = map[string]*Guild{}

func (sock *Socket) getGateway() string {
	data, _ := http.Get("https://discord.com/api/gateway")
	var payload map[string]string
	bytes, _ := io.ReadAll(data.Body)
	_ = json.Unmarshal(bytes, &payload)
	return fmt.Sprintf("%s?v=10&encoding=json", payload["url"])
}

type Socket struct {
	Intent   int
	Memoize  bool
	Presence Presence
}

func (sock *Socket) keepAlive(conn *websocket.Conn, dur int) {
	for {
		_ = conn.WriteJSON(map[string]interface{}{"op": 1, "d": 251})
		beatSent = time.Now().UnixMilli()
		time.Sleep(time.Duration(dur) * time.Millisecond)
	}
}

func (sock *Socket) identify(conn *websocket.Conn, Token string, intent int) {
	type properties struct {
		Os      string `json:"os"`
		Browser string `json:"browser"`
		Device  string `json:"device"`
	}
	type data struct {
		Token      string                 `json:"token"`
		Intents    int                    `json:"intents"`
		Properties properties             `json:"properties"`
		Presence   map[string]interface{} `json:"presence"`
	}
	d := data{
		Token:   Token,
		Intents: intent,
		Properties: properties{
			Os:      "linux",
			Browser: "disgo",
			Device:  "disgo",
		},
		Presence: sock.Presence.Marshal(),
	}
	if sock.Presence.OnMobile {
		d.Properties.Browser = "Discord iOS"
	}
	payload := map[string]interface{}{"op": 2, "d": d}
	_ = conn.WriteJSON(payload)
}

func (sock *Socket) AddHandler(name string, fn interface{}) {
	eventHooks[name] = fn
}

func (sock *Socket) RegistrationQueue(commands ...ApplicationCommand) {
	for _, com := range commands {
		queue = append(queue, com)
	}
}

func registerCommand(com ApplicationCommand, token string, applicationId string) {
	var route string
	data, hook, guildId := com.Marshal()
	if guildId != 0 {
		route = fmt.Sprintf("/applications/%s/guilds/%v/commands", applicationId, guildId)
	} else {
		route = fmt.Sprintf("/applications/%s/commands", applicationId)
	}
	r := MinimalReq("POST", route, data, token)
	d := map[string]interface{}{}
	body, _ := io.ReadAll(r.Request().Body)
	_ = json.Unmarshal(body, &d)
	_, ok := d["id"]
	if ok {
		commandHooks[d["id"].(string)] = hook
	} else {
		log.Fatal(
			fmt.Sprintf("Failed to register command {%s}. Code %s", com.Name, d["message"]))
	}
}

func (sock *Socket) Run(token string) {
	wss := sock.getGateway()
	conn, _, err := websocket.DefaultDialer.Dial(wss, nil)
	if err != nil {
		log.Fatal(err)
	}
	for {
		var wsmsg struct {
			Op       int                    `json:"op"`
			Event    string                 `json:"t"`
			Sequence int                    `json:"s"`
			Data     map[string]interface{} `json:"d"`
		}
		_ = conn.ReadJSON(&wsmsg)
		var r runtime
		r.Sequence = wsmsg.Sequence
		if wsmsg.Event == OnReady {
			ba, _ := json.Marshal(wsmsg.Data)
			_ = json.Unmarshal(ba, &r)
			for _, cmd := range queue {
				go registerCommand(cmd, token, r.Application.Id)
			}
			b = bot.Unmarshal(wsmsg.Data["user"].(map[string]interface{}))
			b.Latency = latency
			isready = true
			b.IsReady = true
			if _, ok := eventHooks[OnReady]; ok {
				go eventHooks[OnReady].(func(bot bot.User))(*b)
			}
		}
		if wsmsg.Op == 10 {
			interval = wsmsg.Data["heartbeat_interval"].(float64)
			sock.identify(conn, token, sock.Intent)
			go sock.keepAlive(conn, int(interval))
		}
		if wsmsg.Op == 11 {
			beatReceived = time.Now().UnixMilli()
			latency = beatReceived - beatSent
			if b != nil {
				b.Latency = latency
			}
		}
		if wsmsg.Op == 7 {
			_ = conn.WriteJSON(map[string]interface{}{
				"op": 6,
				"d": map[string]interface{}{
					"token":      token,
					"session_id": r.SessionId,
					"sequence":   r.Sequence,
				},
			})
		}
		if wsmsg.Op == 9 {
			sock.identify(conn, token, sock.Intent)
		}
		eventHandler(wsmsg.Event, wsmsg.Data)
		if h, ok := eventHooks[OnSocketReceive]; ok {
			handler := h.(func(d map[string]interface{}))
			go handler(wsmsg.Data)
		}
		if wsmsg.Event == "GUILD_CREATE" {
			gld := DataToGuild(wsmsg.Data)
			gld.UnmarshalRoles(wsmsg.Data["roles"].([]interface{}))
			gld.UnmarshalChannels(wsmsg.Data["channels"].([]interface{}))
			guilds[gld.Id] = gld
			if sock.Memoize {
				requestMembers(conn, gld.Id)
			}
		}
		if wsmsg.Event == "GUILD_MEMBERS_CHUNK" {
			isready = true
			id := wsmsg.Data["guild_id"].(string)
			guilds[id].UnmarshalMembers(wsmsg.Data["members"].([]interface{}))
			isready = false
		}
	}
}

func eventHandler(event string, data map[string]interface{}) {
	if isready == true {
		return
	}
	switch event {

	case OnMessageCreate:
		if _, ok := eventHooks[event]; ok {
			eventHook := eventHooks[event].(func(bot bot.User, message Message))
			go eventHook(*b, *DataToMessage(data))
		}

	case OnGuildCreate:
		if _, ok := eventHooks[event]; ok {
			eventHook := eventHooks[event].(func(bot bot.User, guild Guild))
			go eventHook(*b, *DataToGuild(data))
		}

	case OnInteractionCreate:

		i := DataToInteraction(data)

		if _, ok := eventHooks[event]; ok {
			eventHook := eventHooks[event].(func(bot bot.User, ctx Interaction))
			go eventHook(*b, *i)
		}
		switch i.Type {
		case 1:
			// interaction ping
		case 2:
			ctx := CommandContext{
				Id:            i.Id,
				Token:         i.Token,
				ApplicationId: i.ApplicationId,
			}
			if _, ok := commandHooks[i.Data.Id]; ok {
				hook := commandHooks[i.Data.Id].(func(bot bot.User, ctx CommandContext, ops ...SlashCommandOption))
				go hook(*b, ctx, i.Data.Options...)
			}
		case 3:
			factory := CallbackTasks
			cc := DataToComponentContext(data)
			switch cc.Data.ComponentType {
			case 2:
				cb, ok := factory[cc.Data.CustomId]
				if ok {
					callback := cb.(func(b bot.User, ctx ComponentContext))
					go callback(*b, *cc)
				}
			case 3:
				cb, ok := factory[cc.Data.CustomId]
				if ok {
					callback := cb.(func(b bot.User, ctx ComponentContext, values ...string))
					go callback(*b, *cc, cc.Data.Values...)
				}
			}
			tmp, ok := TimeoutTasks[cc.Data.CustomId]
			if ok {
				onTimeoutHandler := tmp[1].(func(b bot.User, i ComponentContext))
				duration := tmp[0].(float64)
				delete(TimeoutTasks, cc.Data.CustomId)
				go scheduleTimeoutTask(duration, *b, *cc, onTimeoutHandler)
			}
		case 4:
			// handle auto-complete interaction
		case 5:
			cc := DataToComponentContext(data)
			callback, ok := CallbackTasks[cc.Data.CustomId]
			if ok {
				go callback.(func(b bot.User, cc ComponentContext))(*b, *cc)
				delete(CallbackTasks, cc.Data.CustomId)
			}
		default:
			log.Println("Unknown interaction type: ", i.Type)
		}
	default:
	}
}

func scheduleTimeoutTask(timeout float64, user bot.User, cc ComponentContext,
	handler func(bot bot.User, cc ComponentContext)) {
	time.Sleep(time.Duration(timeout) * time.Second)
	handler(user, cc)
}

func requestMembers(conn *websocket.Conn, guildId string) {
	_ = conn.WriteJSON(map[string]interface{}{
		"op": 8,
		"d": map[string]interface{}{
			"guild_id": guildId,
			"query":    "",
			"limit":    0,
		},
	})
}