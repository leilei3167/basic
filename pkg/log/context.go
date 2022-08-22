package log

import "context"

type key int

const (
	logContextKey key = iota
)

func WithContext(ctx context.Context) context.Context {
	return std.WithContext(ctx)
}

// WithContext 将logger放入到ctx中进行传递
//将 Logger 添加到 Context 中，并通过 Context 在不同函数间传递，可以使 key-value 在不同函数间传递
//特别适用于controller获取到requestID之后,通过WithValue创建logger后,通过ctx传递,调用链上都可以获取该logger来
//打印日志
func (l *zapLogger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, logContextKey, l)
}

// FromContext 从ctx取出logger,如果没有logger,则创建一个Unknown-Context
func FromContext(ctx context.Context) Logger {
	if ctx != nil {
		logger := ctx.Value(logContextKey)
		if logger != nil {
			return logger.(Logger)
		}
	}
	return WithName("Unknown-Context")
}
