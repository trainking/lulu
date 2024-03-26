/*
lulu是一个游戏服务器框架，支持tcp， kcp，websocket协议。
*/

package lulu

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
)

type (
	// App 游戏服务器应用实现
	App struct {
		listener        network.Listener        // 网络监听
		Config          *Config                 // 系统配置
		SessionManager  *session.SessionManager // 会话管理器
		RouterManager   *RouterManager          // 路由管理器
		exitChan        chan struct{}           // 退出通知
		exitOnce        sync.Once               // 退出单例控制
		modules         []Module                // 模块列表
		connectEvent    SessionEvent            // 连接事件
		disconnectEvent SessionEvent            // 断连事件
	}
)

// New 创建一个服务器的App
func New(config *Config) *App {
	app := new(App)
	app.Config = config
	app.exitChan = make(chan struct{})

	app.init()
	return app
}

// init 对服务器进行初始化
func (app *App) init() {
	var err error

	// 初始化Listener
	lF := network.NewListenerFactory(app.Config.NetWork, app.Config.Address, app.Config.ConnWriteTimeout, app.Config.ConnWriteTimeout)
	if app.Config.TLS != nil {
		cert, err := tls.LoadX509KeyPair(app.Config.TLS.CertFile, app.Config.TLS.KeyFile)
		if err != nil {
			panic(err)
		}
		lF.WithTlsConfig(&tls.Config{
			Certificates: []tls.Certificate{cert},
		})
	}
	if app.Config.KcpMode != "" {
		lF.WithKcpMode(app.Config.KcpMode)
	}
	if app.Config.WebsocketPath != "" {
		lF.WithUpgradePath(app.Config.WebsocketPath)
	}
	app.listener, err = lF.Generate()
	if err != nil {
		panic(err)
	}

	// 初始化会话管理器
	app.SessionManager = session.NewSessionManager()

	// 初始化路由管理器
	app.RouterManager = NewRouterManager()
}

// Route 返回路由管理器
func (app *App) Route() *RouterManager {
	return app.RouterManager
}

// Run 运行服务器
func (app *App) Run(modules ...Module) {
	go func() {
		exitC := make(chan os.Signal, 1)
		signal.Notify(exitC, syscall.SIGINT, syscall.SIGTERM)

		<-exitC
		app.Destory()
		os.Exit(0)
	}()
	defer func() {
		e := recover()
		if e != nil {
			fmt.Printf("painc :%v\n", e)
		}
		app.Destory()
	}()

	// 加入模块
	for _, m := range modules {
		if err := m.OnInit(app); err != nil {
			panic(err)
		}
		m.Route(app)
		app.modules = append(app.modules, m)
	}

	app.run()
}

// run 具体运行的逻辑
func (app *App) run() {
	for {
		select {
		case <-app.exitChan:
			return
		default:
		}

		conn, err := app.listener.Accept()
		if err != nil {
			fmt.Printf("listener error: %v\n", err)
			continue
		}

		go func() {
			s := session.NewSession(conn, app)
			s.Run()

			validTimer := time.NewTimer(time.Duration(app.Config.ValidTimeout) * time.Second)
			select {
			case <-validTimer.C:
				if !s.IsValid() {
					s.Destory()
					return
				}
			case <-s.WaitValid():
			}
			app.SessionManager.Add(s)
		}()
	}
}

// SetConnectEvent 设置连接事件
func (app *App) SetConnectEvent(event SessionEvent) {
	app.connectEvent = event
}

// SetDisconnectEvent 设置断连事件
func (app *App) SetDisconnectEvent(event SessionEvent) {
	app.disconnectEvent = event
}

// Action 向特定玩家触发消息，只可以向玩家触发返回消息或者触发其内部路由
func (app *App) Action(userID uint64, msg proto.Message) {
	// 先获取此玩家是否在在线
	if s, ok := app.SessionManager.Get(userID); !ok {
		return
	} else {
		msgName := msg.ProtoReflect().Descriptor().FullName()
		_r, ok := app.RouterManager.GetInnerRouter(msgName)
		if !ok {
			_, ok := app.RouterManager.GetSendOpCode(msgName)
			if !ok {
				fmt.Printf("%s\tAction Eerror: %v UserID: %v\n", time.Now().Format(time.RFC3339), "no register router", userID)
				return
			}

			if err := s.Send(msg); err != nil {
				fmt.Printf("%s\tAction Send Eerror: %v UserID: %v\n", time.Now().Format(time.RFC3339), err, userID)
			}
			return
		}

		// 内部路由执行
		msgB, _ := proto.Marshal(msg)
		p := network.PackingOpcode(_r.OpCode, msgB)
		go app.asyncHandleMessage(s, _r, p)
	}
}

// asyncHandleMessage 异步处理消息
func (app *App) asyncHandleMessage(s *session.Session, r Router, p network.Packet) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("%s\tasyncHandleMessge Eerror: %v Opcode: %v\n", time.Now().Format(time.RFC3339), e, r.OpCode)
		}
		p.Free()
	}()

	ctx := NewContext(context.Background(), app, s, p.OpCode(), p.Body())
	h := r.Handler
	// 处理消息之前，中间件过滤
	if len(r.Middleware) > 0 {
		for i := len(r.Middleware) - 1; i >= 0; i-- {
			h = r.Middleware[i](h)
		}
	}
	// 处理消息
	if err := h(ctx); err != nil {
		fmt.Printf("%s\tOnMessage Eerror: %v Opcode: %v\n", time.Now().Format(time.RFC3339), err, r.OpCode)
		return
	}
}

// Destory 销毁App
func (app *App) Destory() {
	app.exitOnce.Do(func() {
		// 先销毁模块，倒序销毁
		for i := len(app.modules) - 1; i >= 0; i-- {
			app.modules[i].OnDestory()
		}
		close(app.exitChan)
		app.listener.Close()
	})
}
