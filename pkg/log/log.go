package log

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"sync"
)

// InfoLogger 代表记录非错误信息的能力
type InfoLogger interface {
	// Info 使用时需要指定值的类型,底层不需要反射,性能最高
	Info(msg string, fields ...Field) //example:log.Info("This is a info message", log.Int32("int_key", 10))
	Infof(fomat string, args ...any)  //example;log.Infof("This is a formatted %s message", "info")
	// Infow 使用指定的key-value记录日志
	Infow(msg string, keyAndValues ...any) //example:log.Infow("Message printed with Infow", "X-Request-ID", "fbf54504-64da-4088-9b86-67824a7fb508")

	Enabled() bool
}

// Logger 代表记录日志信息的能力,包含错误信息和其他
type Logger interface {
	InfoLogger

	Debug(msg string, fields ...Field)
	Dubugf(format string, args ...any)
	Debugw(msg string, keysAndValues ...any)

	Warn(msg string, fields ...Field)
	Warnf(format string, args ...any)
	Warnw(msg string, keysAndValues ...any)

	Error(msg string, field ...Field)
	Errorf(format string, args ...any)
	Errorw(format string, keysAndValues ...any)

	Panic(msg string, fields ...Field)
	Panicf(format string, v ...any)
	Panicw(msg string, keysAndValues ...any)

	Fatal(msg string, fields ...Field)
	Fatalf(format string, v ...any)
	Fatalw(msg string, keysAndValues ...any)

	V(level Level) InfoLogger
	Write(p []byte) (n int, err error)

	WithValues(keysAndValues ...any) Logger

	WithName(name string) Logger

	WithContext(ctx context.Context) context.Context

	//Flush 调用底层Core的Sync方法，刷新任何缓冲的日志条目。应用程序应注意在退出前调用Sync。
	Flush()
}

type noopInfoLogger struct{}

func (l *noopInfoLogger) Info(msg string, fields ...Field) {}

func (l *noopInfoLogger) Infof(fomat string, args ...any) {}

func (l *noopInfoLogger) Infow(msg string, keyAndValues ...any) {}

func (l *noopInfoLogger) Enabled() bool { return false }

var disabledInfoLogger = &noopInfoLogger{}

//infoLogger 实现InfoLogger接口,使用zap在指定的整数日志级别记录日志
type infoLogger struct {
	level zapcore.Level
	log   *zap.Logger
}

func (l *infoLogger) Info(msg string, fields ...Field) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(fields...)
	}
}

func (l *infoLogger) Infof(fomat string, args ...any) {
	if checkedEntry := l.log.Check(l.level, fmt.Sprintf(fomat, args)); checkedEntry != nil {
		checkedEntry.Write()
	}
}

func (l *infoLogger) Infow(msg string, keyAndValues ...any) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(handleFields(l.log, keyAndValues)...)
	}
}

func (l *infoLogger) Enabled() bool {
	return true
}

func handleFields(l *zap.Logger, args []any, additional ...zap.Field) []zap.Field {
	if len(args) == 0 {
		return additional
	}
	fields := make([]zap.Field, 0, len(args)/2+len(additional))

	for i := 0; i < len(args); {
		if _, ok := args[i].(zap.Field); ok {
			l.DPanic("strongly-typed Zap Field passed in", zap.Any("zap field", args[i]))
			break
		}
		if i == len(args)-1 {
			l.DPanic("odd number of arguments passed as key-value pairs for logging", zap.Any("ignored key", args[i]))
			break
		}
		//必须确保key为string,将传入的interface分别转换为key和value,并以此构建Field
		key, val := args[i], args[i+1]
		keyStr, isString := key.(string)
		if !isString {
			l.DPanic(
				"non-string key argument passed to logging, ignoring all later arguments",
				zap.Any("invalid key", key),
			)
			break
		}
		fields = append(fields, zap.Any(keyStr, val))
		i += 2
	}

	return append(fields, additional...)

}

var (
	mu  sync.Mutex
	std = New(NewOptions()) //默认的logger
)

// Init 使用特定的options来修改默认的logger
func Init(opts *Options) {
	mu.Lock()
	defer mu.Unlock()
	std = New(opts)
}

// NewLogger 根据zap.logger构建一个实现Logger接口的实例
func NewLogger(l *zap.Logger) Logger {
	return &zapLogger{
		zapLogger: l,
		infoLogger: infoLogger{
			log:   l,
			level: zap.InfoLevel,
		},
	}
}
func ZapLogger() *zap.Logger {
	return std.zapLogger
}

func CheckIntLevel(level int32) bool {
	var lvl zapcore.Level
	if level < 5 {
		lvl = zapcore.InfoLevel
	} else {
		lvl = zapcore.DebugLevel
	}
	checkEntry := std.zapLogger.Check(lvl, "")
	return checkEntry != nil
}

func SugaredLogger() *zap.SugaredLogger {
	return std.zapLogger.Sugar()
}
func StdErrLogger() *log.Logger {
	if std == nil {
		return nil
	}
	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.ErrorLevel); err == nil {
		return l
	}
	return nil
}
func StdInfoLogger() *log.Logger {
	if std == nil {
		return nil
	}
	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.InfoLevel); err == nil {
		return l
	}
	return nil
}

//底层使用zap来记录日志
type zapLogger struct {
	zapLogger  *zap.Logger
	infoLogger //继承infoLogger的打印Info的方法
}

// V 可以通过整型数值来快速指定日志级别打印日志，数值越大，日志级别越高
func V(level Level) InfoLogger { return std.V(level) }

func (l *zapLogger) V(level Level) InfoLogger {
	if l.zapLogger.Core().Enabled(level) {
		return &infoLogger{
			level: level,
			log:   l.zapLogger,
		}
	}
	return disabledInfoLogger
}

func Debug(msg string, fields ...Field) { std.zapLogger.Debug(msg, fields...) }

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.zapLogger.Debug(msg, fields...)
}

func Debugf(format string, v ...any) {
	std.zapLogger.Sugar().Debugf(format, v...)
}

func (l *zapLogger) Dubugf(format string, args ...any) {
	std.zapLogger.Sugar().Debugf(format, args...)
}

func Debugw(msg string, keysAndValues ...any) { std.zapLogger.Sugar().Debugw(msg, keysAndValues...) }

func (l *zapLogger) Debugw(msg string, keysAndValues ...any) {
	l.zapLogger.Sugar().Debugw(msg, keysAndValues...)
}

func Warn(msg string, fields ...Field) { std.zapLogger.Warn(msg, fields...) }

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.zapLogger.Warn(msg, fields...)
}

func Warnf(format string, args ...any) { std.zapLogger.Sugar().Warnf(format, args...) }

func (l *zapLogger) Warnf(format string, args ...any) {
	l.zapLogger.Sugar().Warnf(format, args...)
}

func Warnw(msg string, keysAndValues ...any) { std.zapLogger.Sugar().Warnw(msg, keysAndValues...) }

func (l *zapLogger) Warnw(msg string, keysAndValues ...any) {
	l.zapLogger.Sugar().Warnw(msg, keysAndValues...)
}

func Error(msg string, fields ...Field) { std.zapLogger.Error(msg, fields...) }

func (l *zapLogger) Error(msg string, field ...Field) {
	l.zapLogger.Error(msg, field...)
}

func Errorf(format string, args ...any) { std.zapLogger.Sugar().Errorf(format, args...) }

func (l *zapLogger) Errorf(format string, args ...any) {
	l.zapLogger.Sugar().Errorf(format, args...)
}

func Errorw(format string, keysAndValues ...any) {
	std.zapLogger.Sugar().Errorw(format, keysAndValues...)
}

func (l *zapLogger) Errorw(format string, keysAndValues ...any) {
	l.zapLogger.Sugar().Errorw(format, keysAndValues...)
}

func Panic(msg string, fields ...Field) { std.zapLogger.Panic(msg, fields...) }

func (l *zapLogger) Panic(msg string, fields ...Field) {
	l.zapLogger.Panic(msg, fields...)
}

func Panicf(format string, args ...any) { std.zapLogger.Sugar().Panicf(format, args...) }

func (l *zapLogger) Panicf(format string, v ...any) {
	l.zapLogger.Sugar().Panicf(format, v...)
}

func Panicw(msg string, keysAndValues ...any) { std.zapLogger.Sugar().Panicw(msg, keysAndValues...) }

func (l *zapLogger) Panicw(msg string, keysAndValues ...any) {
	l.zapLogger.Sugar().Panicw(msg, keysAndValues...)
}

func Fatal(msg string, fields ...Field) { std.zapLogger.Fatal(msg, fields...) }

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.zapLogger.Fatal(msg, fields...)
}

func Fatalf(format string, args ...any) { std.zapLogger.Sugar().Fatalf(format, args...) }

func (l *zapLogger) Fatalf(format string, v ...any) {
	l.zapLogger.Sugar().Fatalf(format, v...)
}

func Fatalw(msg string, keysAndValues ...any) { std.zapLogger.Sugar().Fatalw(msg, keysAndValues...) }

func (l *zapLogger) Fatalw(msg string, keysAndValues ...any) {
	l.zapLogger.Sugar().Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) Write(p []byte) (n int, err error) {
	l.zapLogger.Info(string(p))
	return len(p), nil
}
func Info(msg string, fields ...Field) {
	std.zapLogger.Info(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.zapLogger.Info(msg, fields...)
}

func Infof(format string, args ...any) { std.zapLogger.Sugar().Infof(format, args...) }

func (l *zapLogger) Infof(format string, args ...any) {
	l.zapLogger.Sugar().Infof(format, args...)

}

func Infow(msg string, keysAndValues ...any) { std.zapLogger.Sugar().Infow(msg, keysAndValues...) }

func (l *zapLogger) Infow(msg string, keysAndValues ...any) {
	l.zapLogger.Sugar().Infow(msg, keysAndValues...)

}

func WithValues(keysAndValues ...any) Logger {
	return std.WithValues(keysAndValues...)
}

//WithValues 可以返回一个携带指定 key-value 的 Logger，供后面使用。
func (l *zapLogger) WithValues(keysAndValues ...any) Logger {
	newLogger := l.zapLogger.With(handleFields(l.zapLogger, keysAndValues)...)
	return NewLogger(newLogger)
}

func WithName(s string) Logger { return std.WithName(s) }

func (l *zapLogger) WithName(name string) Logger {
	newLogger := l.zapLogger.Named(name)
	return NewLogger(newLogger)
}

func Flush() { std.Flush() }

func (l *zapLogger) Flush() {
	_ = l.zapLogger.Sync()
}

func L(ctx context.Context) *zapLogger {
	return std.L(ctx)
}

// L 方法会从传入的 Context 中提取出 requestID 和 username ，追加到 Logger 中，并返回 Logger
func (l *zapLogger) L(ctx context.Context) *zapLogger {
	lg := l.clone()
	if requestID := ctx.Value(KeyRequestID); requestID != nil {
		lg.zapLogger = lg.zapLogger.With(zap.Any(KeyRequestID, requestID))
	}
	if username := ctx.Value(KeyUsername); username != nil {
		lg.zapLogger = lg.zapLogger.With(zap.Any(KeyUsername, username))
	}

	return lg
}

func (l *zapLogger) clone() *zapLogger {
	copy := *l
	return &copy
}

var _ Logger = (*zapLogger)(nil)

// New 根据opts创建一个logger
func New(opts *Options) *zapLogger {
	if opts == nil {
		opts = NewOptions()
	}

	var zapLevel zapcore.Level //将字符串,如 "debug","warn"等转换为整数level,有错误默认InfoLevel
	if err := zapLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	encodeLevel := zapcore.CapitalLevelEncoder
	// when output to local path, with color is forbidden
	if opts.Format == consoleFormat && opts.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     timeEncoder,
		EncodeDuration: milliSecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	loggerConfig := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       opts.Development,
		DisableCaller:     opts.DisableCaller,
		DisableStacktrace: opts.DisableStacktrace,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         opts.Format,
		EncoderConfig:    encoderConfig,
		OutputPaths:      opts.OutputPaths,
		ErrorOutputPaths: opts.ErrorOutputPaths,
	}

	//构建
	var err error
	l, err := loggerConfig.Build(zap.AddStacktrace(zapcore.PanicLevel), zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}

	logger := &zapLogger{
		zapLogger: l.Named(opts.Name),
		infoLogger: infoLogger{
			level: zap.InfoLevel,
			log:   l,
		},
	}

	zap.RedirectStdLog(l)
	return logger
}
