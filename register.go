package lulu

type (
	// RegisterParams 注册参数
	RegisterParams struct {
		Handler    Handler      // 处理函数
		IsInner    bool         // 是否是内部请求
		Middleware []Middleware // 中间件
		IsNoValid  bool         // 是否无需验证的请求
	}

	// RegisterOptions 注册选项
	RegisterOptions interface {
		ApplyOptions(*RegisterParams)
	}

	// RegisterOptionFunc 注册选项函数
	RegisterOptionFunc func(*RegisterParams)
)

func (f RegisterOptionFunc) ApplyOptions(o *RegisterParams) {
	f(o)
}

// NewRegisterParams 创建注册参数
func NewRegisterParams(opts ...RegisterOptions) *RegisterParams {
	rp := &RegisterParams{}

	for _, opt := range opts {
		opt.ApplyOptions(rp)
	}

	return rp
}

// WithRegisterHandler 注册处理函数
func WithRegisterHandler(h Handler) RegisterOptions {
	return RegisterOptionFunc(func(o *RegisterParams) {
		o.Handler = h
	})
}

// WithRegisterIsInner 是否是内部请求
func WithRegisterIsInner(isInner bool) RegisterOptions {
	return RegisterOptionFunc(func(o *RegisterParams) {
		o.IsInner = isInner
	})
}

// WithRegisterMiddleware 注册中间件
func WithRegisterMiddleware(m Middleware) RegisterOptions {
	return RegisterOptionFunc(func(o *RegisterParams) {
		o.Middleware = append(o.Middleware, m)
	})
}

// WithRegisterIsNoValid 是否无需验证的请求
func WithRegisterIsNoValid(isNoValid bool) RegisterOptions {
	return RegisterOptionFunc(func(o *RegisterParams) {
		o.IsNoValid = isNoValid
	})
}
