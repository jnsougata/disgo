package disgo

// Bot is a function that represents a connection to discord.
func Bot(intent int, cache bool, presence Presence) *connection {
	return &connection{sock: &Socket{Intent: intent, Memoize: cache, Presence: presence}}
}

type connection struct {
	sock *Socket
}

func (con *connection) Run(token string) {
	con.sock.Run(token)
}

func (con *connection) AddCommands(commands ...ApplicationCommand) {
	con.sock.AddToQueue(commands...)
}

func (con *connection) OnSocketReceive(handler func(payload map[string]interface{})) {
	con.sock.AddHandler(OnSocketReceive, handler)
}

func (con *connection) OnMessage(handler func(bot BotUser, message Message)) {
	con.sock.AddHandler(OnMessageCreate, handler)
}

func (con *connection) OnReady(handler func(bot BotUser)) {
	con.sock.AddHandler(OnReady, handler)
}

func (con *connection) OnInteraction(handler func(bot BotUser, ctx *Context)) {
	con.sock.AddHandler(OnInteractionCreate, handler)
}

func (con *connection) OnGuildJoin(handler func(bot BotUser, guild Guild)) {
	con.sock.AddHandler(OnGuildCreate, handler)
}

func (con *connection) OnGuildLeave(handler func(bot BotUser, guild Guild)) {
	con.sock.AddHandler(OnGuildDelete, handler)
}
