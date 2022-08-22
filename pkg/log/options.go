package log

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

/*
Options用于创建zap的logger,供使用
*/
const (
	//配置项的flag名称
	flagLevel             = "log.level"
	flagDisableCaller     = "log.disable-caller"
	flagDisableStacktrace = "log.disable-stacktrace"
	flagFormat            = "log.format"
	flagEnableColor       = "log.enable-color"
	flagOutputPaths       = "log.output-paths"
	flagErrorOutputPaths  = "log.error-output-paths"
	flagDevelopment       = "log.development"
	flagName              = "log.name"

	consoleFormat = "console"
	jsonFormat    = "json"
)

type Options struct {
	//支持输出到多个输出,用逗号分开
	OutputPaths []string `json:"output-paths"       mapstructure:"output-paths"`
	//zap内部错误日志输出路径
	ErrorOutputPaths []string `json:"error-output-paths" mapstructure:"error-output-paths"`
	Level            string   `json:"level"              mapstructure:"level"`
	//日志的输出格式,text或json
	Format string `json:"format"             mapstructure:"format"`
	//是否开启 caller，如果开启会在日志中显示调用日志所在的文件、函数和行号。
	DisableCaller bool `json:"disable-caller"     mapstructure:"disable-caller"`
	//是否在 Panic 及以上级别禁止打印堆栈信息
	DisableStacktrace bool   `json:"disable-stacktrace" mapstructure:"disable-stacktrace"`
	EnableColor       bool   `json:"enable-color"       mapstructure:"enable-color"`
	Development       bool   `json:"development"        mapstructure:"development"`
	Name              string `json:"name"               mapstructure:"name"`
}

func NewOptions() *Options {
	return &Options{
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stdout"},
		Level:             zapcore.InfoLevel.String(),
		Format:            consoleFormat,
		DisableCaller:     false,
		DisableStacktrace: false,
		EnableColor:       false,
		Development:       false,
		Name:              "",
	}
}

// Validate 验证字段
func (o *Options) Validate() []error {
	var errs []error

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil { //将配置中的string转换为level
		errs = append(errs, err)
	}

	format := strings.ToLower(o.Format)
	if format != consoleFormat && format != jsonFormat {
		errs = append(errs, fmt.Errorf("not a valid log format: %q", o.Format))
	}
	return errs
}

// AddFlags 将log的配置项作为flag 加入到某个flagset中
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Level, flagLevel, o.Level, "日志的开关级别")
	fs.BoolVar(&o.DisableCaller, flagDisableCaller, o.DisableCaller, "是否打印调用者信息")
	fs.BoolVar(&o.DisableStacktrace, flagDisableStacktrace, o.DisableStacktrace, "是否禁用panic以下级别的堆栈信息打印")
	fs.StringVar(&o.Format, flagFormat, o.Format, "输出的格式,text或者json")
	fs.BoolVar(&o.EnableColor, flagEnableColor, o.EnableColor, "是否开启颜色输出")
	fs.StringSliceVar(&o.OutputPaths, flagOutputPaths, o.OutputPaths, "设置日志的输出路径")
	fs.StringSliceVar(&o.ErrorOutputPaths, flagErrorOutputPaths, o.ErrorOutputPaths, "设置zap自身的错误的输出路径")
	fs.BoolVar(&o.Development, flagDevelopment, o.Development, "开发模式")
	fs.StringVar(&o.Name, flagName, o.Name, "The name of the logger.")
}

func (o *Options) String() string {
	data, _ := json.Marshal(o)
	return string(data)
}

// Build 根据Options和配置文件创建一个全局的zap的 logger
func (o *Options) Build() error {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		zapLevel = zapcore.InfoLevel //默认info级别
	}

	encodeLevel := zapcore.CapitalLevelEncoder
	if o.Format == consoleFormat && o.EnableColor { //如果开启颜色,并且输出为文本
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}

	//zap logger的配置
	zc := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       o.Development,
		DisableCaller:     o.DisableCaller,
		DisableStacktrace: o.DisableStacktrace,
		Sampling: &zap.SamplingConfig{ //采样策略
			Initial:    100,
			Thereafter: 100,
		},
		//自定义编码输出的选项
		Encoding: o.Format,
		EncoderConfig: zapcore.EncoderConfig{
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
			EncodeName:     zapcore.FullNameEncoder,
		},
		OutputPaths:      o.OutputPaths,
		ErrorOutputPaths: o.ErrorOutputPaths,
	}

	logger, err := zc.Build(zap.AddStacktrace(zapcore.PanicLevel)) //只在panicLevel打印堆栈
	if err != nil {
		return err
	}
	zap.RedirectStdLog(logger.Named(o.Name)) //增加logger的名字
	zap.ReplaceGlobals(logger)               //将zap全局的logger替换为我们自己创建的
	return nil
}
