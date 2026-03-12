# Lulu 使用文档

Lulu 是一个轻量级的 Go 语言游戏服务器框架，支持 TCP、KCP 和 WebSocket 协议。它采用模块化设计，易于扩展和维护。

## 1. 快速开始

### 1.1 安装
确保你的项目已初始化 Go Modules：
```bash
go mod init your_project
go get github.com/trainking/lulu
```

### 1.2 配置文件
在项目根目录创建 `configs/lulu.yaml`：
```yaml
Version: "0.0.1"
Address: "127.0.0.1:8007"
Network: "websocket" # 支持 tcp, kcp, websocket
WebsocketPath: "/ws"
ConnMax: 1000
ValidTimeout: 10 # 连接后未验证身份的超时时间（秒）
HeartLimit: 100 # 每分钟消息频率限制
```

### 1.3 启动服务器
```go
package main

import (
    "github.com/trainking/lulu"
)

func main() {
    // 加载配置
    config := lulu.LoadDefaultAppConfig()
    app := lulu.New(config)

    // 运行并加载模块
    app.Run(
        // 你的模块列表
    )
}
```

## 2. 模块 (Module)

Lulu 的核心是模块化。每个功能逻辑应当封装在一个 `Module` 中。

### 2.1 定义模块
```go
type MyModule struct {}

func (m *MyModule) Name() string { return "my_module" }
func (m *MyModule) OnInit(app *lulu.App) error { return nil }
func (m *MyModule) OnDestroy() {}
func (m *MyModule) Route(app *lulu.App) {
    // 在这里注册路由
}
```

## 3. 路由注册

Lulu 提供三种类型的路由：

### 3.1 外部路由 (External Router)
处理客户端发往服务器的消息。
```go
app.Route().Register(&msg.LoginReq{}, 1001, lulu.WithRegisterHandler(m.OnLogin))
```

### 3.2 内部路由 (Internal Router)
处理服务器内部逻辑转发或跨服务消息。
```go
app.Route().Register(&msg.InternalMsg{}, 2001, 
    lulu.WithRegisterHandler(m.OnInternal),
    lulu.WithRegisterIsInner(true),
)
```

### 3.3 返回路由 (Return Router)
定义服务器发往客户端的消息编号。
```go
app.Route().Register(&msg.LoginAck{}, 1002)
```

## 4. 处理器 (Handler)

Handler 是处理具体业务逻辑的函数。

### 4.1 函数签名
```go
func (m *MyModule) OnLogin(ctx lulu.Context) error {
    req := &msg.LoginReq{}
    if err := ctx.Bind(req); err != nil {
        return err
    }
    
    // 业务逻辑...
    
    // 获取 Session
    s := ctx.Session()
    
    // 设置用户ID（标记为有效连接）
    s.SetUserID(123)
    
    // 发送响应
    return s.Send(&msg.LoginAck{Result: true})
}
```

## 5. 中间件 (Middleware)

支持在 Handler 执行前后插入自定义逻辑。

### 5.1 定义中间件
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

### 5.2 注册中间件
```go
app.Route().Register(&msg.DataReq{}, 1003, 
    lulu.WithRegisterHandler(m.OnData),
    lulu.WithRegisterMiddleware(MyMiddleware()),
)
```

## 6. 消息推送

可以通过 `App` 实例主动向玩家推送消息：

- **Action**: 通过 `UserID` 推送（会自动查找在线 Session）。
  ```go
  app.Action(123, &msg.Notify{Content: "Hello"})
  ```
- **Call**: 直接通过 `Session` 对象推送。
  ```go
  app.Call(session, &msg.Notify{Content: "Hello"})
  ```

## 7. 连接事件处理

可以监听连接和断开事件：
```go
app.SetConnectEvent(func(s *session.Session) {
    fmt.Printf("User connected: %d\n", s.ID)
})

app.SetDisconnectEvent(func(s *session.Session) {
    fmt.Printf("User disconnected: %d (UserID: %d)\n", s.ID, s.UserID)
})
```

## 8. 安全特性

- **消息长度限制**: 默认最大 64MB。
- **验证超时**: 客户端连接后需在 `ValidTimeout` 时间内调用 `s.SetUserID()`，否则会被强制断开。
- **洪水攻击防护**: 通过配置 `HeartLimit` 限制每分钟单客户端最大消息数。
