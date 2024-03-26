package lulu

// Module 模块的接口定义
type Module interface {
	// Name 返回模块的一个唯一名字
	Name() string

	// OnInit 初始化模块
	OnInit(app *App) error

	//OnDestory 销毁模块
	OnDestory()

	// Route 注册路由
	Route(app *App)
}
