package lulu

// Mideeleware 中间件定义
type Middleware func(next Handler) Handler

// MiddlewareValidSession 验证会话有效的中间件
func MiddlewareValidSession() Middleware {
	return func(next Handler) Handler {
		return func(ctx Context) error {
			if !ctx.Session().IsValid() {
				return ErrSessionInvalid
			}
			return next(ctx)
		}
	}
}
