# lulu

[English README](./README.md)

lulu 是一个轻量级的游戏服务器开发框架。名字来源于 LOL 中的英雄——Lulu。

## 特性

* 支持多传输协议：TCP / KCP / WebSocket
* 模块化架构
* 内置 TLS 支持
* 洪水攻击防护（单连接消息速率限制）
* 内置客户端 SDK，可连接 lulu 服务器
* 中间件支持

## 快速开始

在项目根目录下创建配置文件 `configs/lulu.yaml`：

```yaml
# 服务版本号
Version: "0.0.1"
# 监听地址
Address: "127.0.0.1:8007"
# 传输层协议: tcp, kcp, websocket
Network: "websocket"
# WebSocket 升级路径
WebsocketPath: "/ws"
# KCP 模式: normal（普通模式）或 fast（极速模式），默认 fast
KcpMode: "fast"
# 每个连接的读超时（也作为心跳超时），秒为单位，默认 10
ConnReadTimeout: 10
# 每个连接的写超时，秒为单位，默认 5
ConnWriteTimeout: 5
# 最大连接数，默认 10000
ConnMax: 10000
# 连接成功后验证身份的超时时间，超时未验证则断开，秒为单位，默认 10
ValidTimeout: 10
# 心跳包限制：每分钟单连接最大消息数，默认 100
HeartLimit: 100
# 服务加密使用的共享密钥
Password: "66014775009e4106"
# 外部访问的 URL
OutUrl: "ws://127.0.0.1:8007"

# TLS 配置（可选，不需要时删除此段）
TLS:
  CertFile: "certs/server.crt"
  KeyFile:  "certs/server.key"
```

然后在 main 中启动服务：

```go
func main() {
    app := lulu.New(lulu.LoadDefaultAppConfig())

    app.Run(
        gate.Module(),
        game.Module(),
    )
}
```

## 模块

模块是 lulu 的核心。每个模块必须实现 `Module` 接口：

```go
type Module interface {
    Name() string
    OnInit(app *App) error
    OnDestroy()
    Route(app *App)
}
```

一个典型的项目通常需要三个模块：
- **gate**：负责玩家接入——登录验证、断线重连等
- **game**：游戏逻辑
- **wait**：负责优雅关闭

模块按注册顺序初始化，按逆序销毁（最后注册的最先销毁）。

### 模块示例

```go
var Module = func() lulu.Module {
    return new(M)
}

type M struct {}

func (m *M) Name() string                    { return "game" }
func (m *M) OnInit(app *lulu.App) error      { return nil }
func (m *M) OnDestroy()                      {}
func (m *M) Route(app *lulu.App) {
    // 在此注册路由
}
```

## 路由

在每个模块的 `Route` 方法中，通过 `app.Route().Register()` 注册路由。lulu 中有三种路由：

### 外部路由（客户端 → 服务器）

处理客户端发往服务器的消息：

```go
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq,
    lulu.WithRegisterHandler(m.AuthReq),
    lulu.WithRegisterIsNoValid(true),  // 登录请求无需验证 session
)
```

### 内部路由（服务器内部消息转发）

处理服务器内部逻辑或跨服务消息：

```go
app.Route().Register(&msg.InternalMsg{}, msg.OpcodeInternal,
    lulu.WithRegisterHandler(m.OnInternal),
    lulu.WithRegisterIsInner(true),
)
```

### 返回路由（服务器 → 客户端）

返回路由只需指定消息类型和 opcode，无需 Handler：

```go
app.Route().Register(&msg.AuthAck{}, msg.OpcodeAuthAck)
```

## Handler

Handler 是处理具体业务逻辑的函数，必须遵循以下签名：

```go
func HandlerName(ctx lulu.Context) error
```

在 Handler 中：
- `ctx.Bind(msg)` — 将请求体反序列化为 protobuf 消息。
- `ctx.Session()` — 获取当前玩家的 `*session.Session`。
- `ctx.App()` — 获取 `*App` 实例。
- `ctx.GetOpCode()` — 获取当前请求的 opcode。

示例：

```go
func (m *M) OnLogin(ctx lulu.Context) error {
    req := &msg.LoginReq{}
    if err := ctx.Bind(req); err != nil {
        return err
    }
    s := ctx.Session()
    s.SetUserID(req.UserID)   // 标记 session 为有效
    return s.Send(&msg.LoginAck{Result: true})
}
```

## 中间件

中间件用于在 Handler 执行前后插入自定义逻辑：

```go
func MyMiddleware() lulu.Middleware {
    return func(next lulu.Handler) lulu.Handler {
        return func(ctx lulu.Context) error {
            // 前置逻辑
            err := next(ctx)
            // 后置逻辑
            return err
        }
    }
}
```

在路由中注册中间件：

```go
app.Route().Register(&msg.DataReq{}, 1003,
    lulu.WithRegisterHandler(m.OnData),
    lulu.WithRegisterMiddleware(MyMiddleware()),
)
```

内置的 `MiddlewareValidSession()` 中间件会自动应用到所有路由上（检查 session 是否已通过验证），除非显式设置 `WithRegisterIsNoValid(true)`。登录/认证路由应使用 `IsNoValid` 跳过此检查。

## 消息推送

- **`app.Action(userID, msg)`** — 根据 userID 查找在线 session 并推送消息。如果用户不在线，消息将被静默忽略。
- **`app.Call(session, msg)`** — 直接向指定 session 推送消息。如果该消息注册了内部路由，则执行内部 Handler 而非发送给客户端。

```go
// 向用户 123 推送通知
app.Action(123, &msg.Notify{Content: "Hello"})

// 直接通过 session 发送
app.Call(session, &msg.Notify{Content: "Hello"})
```

## 连接事件

```go
app.SetConnectEvent(func(s *session.Session) {
    fmt.Printf("用户连接: %d\n", s.ID)
})

app.SetDisconnectEvent(func(s *session.Session) {
    fmt.Printf("用户断开: %d (UserID: %d)\n", s.ID, s.UserID)
})
```

## 客户端

lulu 提供了 `Client` 类型，方便在 Go 代码中连接 lulu 服务器：

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

// 发送消息
client.Send(1001, &msg.LoginReq{Username: "test"})

// 接收消息
for packet := range client.Receive() {
    // 处理消息包
}
```

支持的网络类型：`network.TcpNet`（`"tcp"`）、`network.KcpNet`（`"kcp"`）、`network.WebSocketNet`（`"websocket"`）。

## 协议

定长包头 + 变长包体，采用大端字节序：

```
 |-----------------------------message-----------------------------------------|
 |----------------------Header------------------|------------Body--------------|
 |------Body Length-------|--------Opcode-------|------------Body--------------|
 |---------uint16---------|---------uint16------|------------bytes-------------|
 |-----------2------------|----------2----------|-----------len(Body)----------|
```

## 安全特性

- **消息大小限制**：单包最大 64 MB，防止内存溢出攻击。
- **验证超时**：客户端须在 `ValidTimeout` 秒内通过 `session.SetUserID()` 完成身份验证，否则强制断开。
- **洪水攻击防护**：通过 `HeartLimit` 限制单连接每分钟最大消息数。默认 100，设为 0 关闭此功能。
- **TLS 加密**：在配置文件中添加 `TLS` 段即可为任意协议启用传输层加密。

## 贡献者

- xiaoye
