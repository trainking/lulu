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