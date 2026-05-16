# lulu

[中文文档](./REAEMD_zh.md)

lulu is a game server development framework. The name comes from the LOL hero - Lulu.

## Features

* Multiple transport protocols: TCP / KCP / WebSocket
* Modular architecture
* Built-in TLS support
* Flood protection (per-connection message rate limiting)
* Client SDK for connecting to lulu servers
* Middleware support

## Quick Start

Create a config file at `configs/lulu.yaml` under your project root:

```yaml
# Server version
Version: "0.0.1"
# Listen address
Address: "127.0.0.1:8007"
# Transport protocol: tcp, kcp, websocket
Network: "websocket"
# WebSocket upgrade path
WebsocketPath: "/ws"
# KCP mode: normal (普通模式) or fast (极速模式), default fast
KcpMode: "fast"
# Read timeout per connection in seconds (also acts as heartbeat timeout), default 10
ConnReadTimeout: 10
# Write timeout per connection in seconds, default 5
ConnWriteTimeout: 5
# Maximum connections, default 10000
ConnMax: 10000
# Authentication timeout in seconds — disconnected if not authenticated within this window, default 10
ValidTimeout: 10
# Heartbeat rate limit: max messages per minute per connection, default 100
HeartLimit: 100
# Shared secret for encryption
Password: "66014775009e4106"
# External access URL
OutUrl: "ws://127.0.0.1:8007"

# TLS configuration (optional, remove if not needed)
TLS:
  CertFile: "certs/server.crt"
  KeyFile:  "certs/server.key"
```

Then start the server:

```go
func main() {
    app := lulu.New(lulu.LoadDefaultAppConfig())

    app.Run(
        gate.Module(),
        game.Module(),
    )
}
```

## Modules

Modules are the core of lulu. Each module implements the `Module` interface:

```go
type Module interface {
    Name() string
    OnInit(app *App) error
    OnDestroy()
    Route(app *App)
}
```

A typical project has three modules:
- **gate**: Handles player connections — login, reconnection, etc.
- **game**: Game logic.
- **wait**: Graceful shutdown coordination.

Modules are initialized in registration order, and destroyed in reverse order.

### Example module

```go
var Module = func() lulu.Module {
    return new(M)
}

type M struct {}

func (m *M) Name() string                    { return "game" }
func (m *M) OnInit(app *lulu.App) error      { return nil }
func (m *M) OnDestroy()                      {}
func (m *M) Route(app *lulu.App) {
    // register routes here
}
```

## Routing

Routes are registered inside each module's `Route` method via `app.Route().Register()`. There are three kinds:

### External routes (client → server)

```go
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq,
    lulu.WithRegisterHandler(m.AuthReq),
    lulu.WithRegisterIsNoValid(true),  // skip session-valid check for auth
)
```

### Internal routes (server-internal message forwarding)

```go
app.Route().Register(&msg.InternalMsg{}, msg.OpcodeInternal,
    lulu.WithRegisterHandler(m.OnInternal),
    lulu.WithRegisterIsInner(true),
)
```

### Return routes (server → client)

Return routes only need the message type and opcode — no handler:

```go
app.Route().Register(&msg.AuthAck{}, msg.OpcodeAuthAck)
```

## Handlers

Every handler must match:

```go
func HandlerName(ctx lulu.Context) error
```

Inside a handler:
- `ctx.Bind(msg)` — deserializes the request body into the given protobuf message.
- `ctx.Session()` — returns the player's `*session.Session`.
- `ctx.App()` — returns the `*App` instance.
- `ctx.GetOpCode()` — returns the opcode of the current request.

Example:

```go
func (m *M) OnLogin(ctx lulu.Context) error {
    req := &msg.LoginReq{}
    if err := ctx.Bind(req); err != nil {
        return err
    }
    s := ctx.Session()
    s.SetUserID(req.UserID)   // mark session as valid
    return s.Send(&msg.LoginAck{Result: true})
}
```

## Middleware

Middleware wraps handlers for pre/post processing. Each middleware is a function:

```go
func MyMiddleware() lulu.Middleware {
    return func(next lulu.Handler) lulu.Handler {
        return func(ctx lulu.Context) error {
            // pre-processing
            err := next(ctx)
            // post-processing
            return err
        }
    }
}
```

Register middleware on a route:

```go
app.Route().Register(&msg.DataReq{}, 1003,
    lulu.WithRegisterHandler(m.OnData),
    lulu.WithRegisterMiddleware(MyMiddleware()),
)
```

A built-in `MiddlewareValidSession()` is automatically applied to all routes unless `WithRegisterIsNoValid(true)` is set. Use `IsNoValid` for login/auth routes where the session is not yet validated.

## Pushing Messages to Players

- **`app.Action(userID, msg)`** — Looks up the session by userID and sends the message. If the user is not online, the message is silently dropped.
- **`app.Call(session, msg)`** — Sends a message directly to a specific session. If the message has an internal route registered, that handler is invoked instead of sending to the client.

```go
// Push a notification to user 123
app.Action(123, &msg.Notify{Content: "Hello"})

// Send via session directly
app.Call(session, &msg.Notify{Content: "Hello"})
```

## Connection Events

```go
app.SetConnectEvent(func(s *session.Session) {
    fmt.Printf("User connected: %d\n", s.ID)
})

app.SetDisconnectEvent(func(s *session.Session) {
    fmt.Printf("User disconnected: %d (UserID: %d)\n", s.ID, s.UserID)
})
```

## Client

lulu ships a `Client` type for connecting to lulu servers from Go code:

```go
config := &network.Config{
    Addr:         "127.0.0.1:8007",
    ReadTimeout:  10,
    WriteTimeout: 5,
}

client, err := lulu.NewClient(network.WebSocketNet, config)
if err != nil {
    panic(err)
}
defer client.Close()

// Send a message
client.Send(1001, &msg.LoginReq{Username: "test"})

// Receive messages
for packet := range client.Receive() {
    // handle packet
}
```

Supported network types: `network.TcpNet` (`"tcp"`), `network.KcpNet` (`"kcp"`), `network.WebSocketNet` (`"websocket"`).

## Protocol

Fixed-length header + variable-length body, big-endian byte order:

```
 |-----------------------------message-----------------------------------------|
 |----------------------Header------------------|------------Body--------------|
 |------Body Length-------|--------Opcode-------|------------Body--------------|
 |---------uint16---------|---------uint16------|------------bytes-------------|
 |-----------2------------|----------2----------|-----------len(Body)----------|
```

## Security

- **Message size limit**: Maximum 64 MB per packet to prevent OOM attacks.
- **Authentication timeout**: Clients must authenticate within `ValidTimeout` seconds (call `session.SetUserID()`), or the connection is dropped.
- **Flood protection**: Per-connection message rate limiting via `HeartLimit` (max messages per minute). Defaults to 100; set to 0 to disable.
- **TLS**: Configure the `TLS` section in your config to enable transport-layer encryption for any protocol.

## Contributor

- xiaoye
