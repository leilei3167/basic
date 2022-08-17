package app

// CliOptions 是能够从命令行获取参数的抽象
type CliOptions interface {
	// Flags
	Flags() (fss NamedFlagSets)
	// Validate 用于验证命令行选项的输入,错误将以切片的形式被收集
	Validate() []error
}

// ConfigurableOptions 是能够从配置文件获取参数的抽象
type ConfigurableOptions interface {
	// ApplyFlags 从命令行参数或者配置文件解析参数
	ApplyFlags() []error
}

// CompleteableOptions 配置自动补全的抽象接口
type CompleteableOptions interface {
	Complete() error
}

// PrintableOptions 能够被打印的抽象接口
type PrintableOptions interface {
	String() string
}
