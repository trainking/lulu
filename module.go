package lulu

type Module interface {
	// OnInit 初始化模块
	OnInit(app *App) error

	//OnDestory 销毁模块
	OnDestory()

	// Route 注册路由
	Route(app *App)
}
