# lulu

lulu是一个游戏服务器的开发框架。名字来源于LOL中的英雄--lulu。

## 特性

* 支持多协议：tcp/kcp/websocket
* 模块化
* 容易上手

## 快速开始

lulu需要一份配置文件，来配置服务器的一些特定事项。默认存放在项目根路径下的`configs/lulu.yaml`中，他的配置如下:

```yaml
# 以下为游戏配置，即app.AppConfig读取
# 服务的版本号
Version: "0.0.1"
# 监听地址
Address: "127.0.0.1:8007"
# 传输层协议，tcp, kcp，websocket
Network: "websocket"
# websocket时使用升级路径
WebsocketPath: "/ws"
# kcp模式，nomarl 普通模式 fast 极速模式；默认极速模式
KcpMode: "fast"
# 每个连接的读超时(等于客户端心跳的超时)，秒为单位
ConnReadTimeout: 10
# 每个连接的写超时，秒为单位
ConnWriteTimeout: 5
# 最大连接数
ConnMax: 1000
# 连接成功后，多久未验证身份，则断开，秒为单位
ValidTimeout: 10
# 心跳包限制数量, 每分钟不能超过的数量
HeartLimit: 100
# 服务加密使用的密码
Password: "66014775009e4106"
# 外部访问的url
OutUrl: "ws://127.0.0.1:8007"
```

你可以根据自己的项目情况，做出一下变更。配置文件增加之后，在main中，就可以使用如下代码开始游戏服务:

```golang
func main() {
	// 从默认的configs/game.yaml中读取配置
	app := lulu.New(lulu.LoadDefaultAppConfig())

	app.Run(
		gate.Module(),
		game.Module(),
	)
}
```

## 模块

模块是lulu的核心。lulu依靠module来组织代码，每个module代表一个功能模块。一般情况下，我们需要三个模块:

- gate：负责玩家服务接入，如登录验证，断线重连等等
- game：负责游戏逻辑
- wait: 负责关闭时，优雅的销毁退出

你也可以根据项目的情况，拆分更细的模块。每一个模块都是一个独立的包，其中必须要有一个实现`Module`接口的结构体。推荐的实现结构如下:

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

模块的初始化，在lulu启动时，会调用`OnInit`方法，销毁时，会调用`OnDestory`方法。需要注意的时，二者的顺序是相反的，初始化时，是先注册的先初始化，销毁时则是相反，最后注册的模块，最先销毁。

### 路由

模块的`Route`方法，用来注册路由。lulu推荐使用根据模块来管理路由，而不是将路由集中在一处注册。lulu中存在三种路由，分别是：

- 外部路由：也就是请求路由，即玩家发送的操作指令消息
- 内部路由：内部路由则是游戏服务内部，根据当前事件，向指定玩家执行特定操作的消息路由
- 返回路由：返回路由，是服务器向玩家发送消息的返回路由

它们都是使用`app.Route().Register()`方法注册。除了共有的opcode参数和消息结构体外，需要通过`RegiserOptions`来指定它们的区分:

**外部路由**

```golang
// 通过RegisterOptions指定opcode和是否启用验证中间件
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq, lulu.WithRegisterHandler(m.AuthReq), lulu.WithRegisterIsNoValid(true))
```

**内部路由**

```golang
// 通过RegisterOptions指定opcode和指定是内部路由
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq, lulu.WithRegisterHandler(m.AuthReq), lulu.WithRegisterIsInner(true))
```


**返回路由**

```golang
// 无需指定额外RegisterOptions
app.Route().Register(&msg.AuthReq{}, msg.OpcodeAuthReq)
```

### Handler

lulu中，Handler是对事件响应的抽象。更具发送者的不同，lulu中存在两种路由：**OuterHandler**和**InnerHandler**。

- OuterHandler: 对外部路由的处理
- InnerHandler: 对内部路由的处理

每个Handler函数，必须必须遵循如下结构：

```golang
func HandlerName(c lulu.Context) error
```

对Handler的调用，外部路由来自客户端的请求，无需多言。而内部路由，则采用App的两个方法调用, 分别是`Action`和`Call`。二者的区别是，Call直接通过已知的Session，向其发送消息，而是通过UserID，从有效连接集合中，查找发送。

## 贡献者

- xiaoye