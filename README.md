# lulu

[中文README](./REAEMD_zh.md)

lulu is a game server development framework. The name comes from the LOL hero - Lulu.

## Features

* Supports multiple protocols: tcp/kcp/websocket
* Modularization
* Easy to use

## Quick Start

Lulu needs a configuration file to configure some specific things about the server. By default, it is stored in'configs/lulu.yaml 'under the project root path. Its configuration is as follows:

```yaml
# The following is the game configuration, that is, app. AppConfig read
# version number
Version: "0.0.1"
# Listening address
Address: "127.0.0.1:8007"
# Transport Layer Protocol，tcp, kcp，websocket
Network: "websocket"
# Upgrade path for websocket
WebsocketPath: "/ws"
# KCP mode, nomarl normal mode fast Quick Mode; default Quick Mode
KcpMode: "fast"
# Read timeout per connection (equal to client side heartbeat timeout) in seconds
ConnReadTimeout: 10
# Write timeout per connection, in seconds
ConnWriteTimeout: 5
# Maximum number of connections
ConnMax: 1000
# After the connection is successful, how long does it take to verify the identity, then it will be disconnected in seconds
ValidTimeout: 10
# Heartbeat limit, the number that cannot be exceeded per minute
HeartLimit: 100
# Password used for service encryption
Password: "66014775009e4106"
# Url for external access
OutUrl: "ws://127.0.0.1:8007"
```

You can make changes according to your project situation. After the configuration file is added, in main, you can use the following code to start the game service:

```golang
func main() {
	// read configs/game.yaml
	app := lulu.New(lulu.LoadDefaultAppConfig())

	app.Run(
		gate.Module(),
		game.Module(),
	)
}
```

## Modules

Modules are the core of lulu. Lulu relies on modules to organize code, and each module represents a functional module. In general, we need three modules:

- gate: The gate module is the entry of the game server, which is responsible for accepting connections from the client and forwarding them to the game module.
- game: The game module is responsible for processing the game logic.
- wait: Responsible for closing when elegantly destroying exits.

You can also split smaller modules depending on the project. Each module is a separate package, which must have a structure that implements the'Module 'interface：

```golang
var Module = func() lulu.Module {
	return new(M)
}

type M struct {
}

func (m *M) Name() string {
	return "game"
}
func (m *M) OnInit(app *lulu.App) error {
	return nil
}
func (m *M) OnDestory() {
}
func (m *M) Route(app *lulu.App) {

}
```

The initialization of the module will call the "OnInit" method when Lulu is started, and the "OnDestory" method will be called when destroyed. It should be noted that the order of the two is opposite. During initialization, it is the first to register first. Initialization is the opposite when destroyed. The last registered module is destroyed first.

### Router

The `Route` method of the module used to register routes. Lulu recommends managing routes based on modules instead of registering routes in one place. There are three routes in Lulu, namely:

- External routing: that is, request routing, that is, the operation instruction message sent by the player.
- Internal routing: Internal routing is the message routing within the game service that performs specific operations to the specified player according to the current event.
- Return routing: Return route, which is the return route for the server to send messages to the player.

They are all registered using the `app.Route().Register()` method. In addition to the common opcode parameters and message structure, you need to specify their differentiation through `RegiserOptions`:

**External routing**

```golang
// Specify opcode and whether to enable authentication middleware with RegisterOptions
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq, lulu.WithRegisterHandler(m.AuthReq), lulu.WithRegisterIsNoValid(true))
```

**Internal routing**

```golang
// Specify opcode and specify internal routes through RegisterOptions
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq, lulu.WithRegisterHandler(m.AuthReq), lulu.WithRegisterIsInner(true))
```

**Return routing**

```golang
// No additional RegisterOptions required
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq)
```

### Handler

In Lulu, the Handler is an abstraction of event response. More different from the sender, there are two routes in Lulu: **OuterHandler** and **InnerHandler**.

- OuterHandler: Handling of external routes
- InnerHandler: Handling of internal routes

Each Handler function must adhere to the following structure:

```golang
func HandlerName(c lulu.Context) error
```

For the call to the Handler, the external route is the request from the client, without further ado. The internal route uses the two method calls of the App, namely `Action` and `Call`. The difference between the two is that the Call directly sends messages to the known Session, but through the UserID, it finds and sends from the set of valid connections.

## Contributor

- xiaoye