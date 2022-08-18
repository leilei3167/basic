package app

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/leilei3167/basic/pkg/base/util/term"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	progressMessage = color.GreenString("==>")
)

// App 是命令行程序的一个通用结构,对于 API 服务和非 API 服务来说，它们的启动流程基本一致,都会有以下3步
//1.应用框架的搭建 2.应用初始化 3.服务启动
//
//其中应用框架搭建这一步可以说是任何程序都有的,这一步
//分为3个内容:
//1. 命令行程序;命令行程序需要实现诸如应用描述、help、参数校验等功能。根据需要，
//还可以实现命令自动补全、打印命令行参数等高级功能。
//2. 命令行参数解析: 用来启动时指定命令行参数,控制应用行为(能够覆盖配置文件中相同的选项)
//3. 配置文件解析: 要能够支持不同格式的配置文件
//这3点跟具体业务关系不大,几乎所有程序都有这个需求,因此,这部分可以抽象为一个统一的,可复用的框架
type App struct {
	basename    string //后续程序要生成的二进制文件名称
	name        string
	description string
	options     CliOptions
	runFunc     RunFunc
	silence     bool
	noVersion   bool
	noConfig    bool
	commands    []*Command           //命令
	args        cobra.PositionalArgs //此处是非命令行选项参数的验证方式(cobra已内置多种)
	cmd         *cobra.Command       //主命令
}

type Option func(*App)

func WithOptions(opt CliOptions) Option {
	return func(a *App) {
		a.options = opt
	}
}

type RunFunc func(basename string) error

func WithRunFunc(run RunFunc) Option {
	return func(a *App) {
		a.runFunc = run
	}
}

func WithDescription(desc string) Option {
	return func(a *App) {
		a.description = desc
	}
}

func WithSilence() Option {
	return func(a *App) {
		a.silence = true
	}
}
func WithNoVersion() Option {
	return func(a *App) {
		a.noVersion = true
	}
}

func WithNoConfig() Option {
	return func(a *App) {
		a.noConfig = true
	}
}

func WithValidArgs(args cobra.PositionalArgs) Option {
	return func(a *App) {
		a.args = args
	}
}

func WithDefauldValidArgs() Option {
	return func(a *App) {
		a.args = func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q 不支持任何非选项参数,输入:%q", cmd.CommandPath(), arg)
				}
			}
			return nil
		}
	}
}

// NewApp 使用选项模式创建App结构
func NewApp(name string, basename string, opts ...Option) *App {
	a := &App{
		name:      name,
		basename:  basename,
		noVersion: true,
	}

	for _, opt := range opts {
		opt(a)
	}
	//根据传入的实例,构建命令行程序
	a.buildCommand()
	return a
}

func (a *App) buildCommand() {
	//1.构建根命令
	cmd := cobra.Command{
		Use:           formatBasename(a.basename),
		Short:         a.name,
		Long:          a.description,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          a.args,
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().SortFlags = true //将选项排序,获得一个pflag.FlagSet
	InitFlags(cmd.Flags())       //设置flag的转换,以及兼容标准库

	//读取是否加入了子命令,如果有,则进行构建
	if len(a.commands) > 0 {
		for _, command := range a.commands {
			cmd.AddCommand(command.cobraCommand())
		}
		//如果有子命令的话,增加额外一个help命令,以查看子命令列表
		cmd.SetHelpCommand(helpCommand(formatBasename(a.basename)))
	}

	//设置程序的入口,程序最终会运行此处,此函数会先处理合并形成应用程序可用的配置项
	if a.runFunc != nil {
		cmd.RunE = a.runCommand
	}

	//2.将传入的option注册为分类的flagSet,并分组存入到命令中
	var namedFlagSets NamedFlagSets
	if a.options != nil {
		namedFlagSets = a.options.Flags()
		for _, set := range namedFlagSets.FlagSets {
			cmd.Flags().AddFlagSet(set) //将分组的flag绑定到主命令中
		}
	}
	globalFlags := namedFlagSets.FlagSet("global")
	//再根据实际的配置决定是否额外添加一些全局选项
	if !a.noVersion {

	}

	//3.处理配置文件,由viper来进行解析
	if !a.noConfig {
		//默认为false,即要提供配置文件,则增加一个全局的选项,指定配置文件名称
		addConfigFlag(a.basename, globalFlags)
	}
	//添加帮助选项
	AddGlobalHelpFlags(globalFlags, cmd.Name())
	//在将global分组加到cmd上
	cmd.Flags().AddFlagSet(globalFlags)

	//设置自定义的帮助页面,此处要实现flagset的分类打印
	addCmdTemplate(&cmd, namedFlagSets)

	a.cmd = &cmd
}

// Run 执行构建好的程序,会按顺序执行注册到cobra.Command中的运行函数
func (a *App) Run() {
	if err := a.cmd.Execute(); err != nil {
		fmt.Printf("%v %v\n", color.RedString("Error:", err))
		os.Exit(1)
	}
}

func (a *App) Command() *cobra.Command {
	return a.cmd
}

//此处会将解析的Flags的值和之前读取的配置文件进行合并,形成最终的应用配置,执行指定的
//运行入口
func (a *App) runCommand(cmd *cobra.Command, args []string) error {
	printWorkingDir()
	PrintFlags(cmd.Flags())
	if !a.noVersion {
		//TODO打印版本
	}
	//noConfig为true代表不提供配置文件,否则代表有配置文件,则将选项参数值和配置文件值合并
	if !a.noConfig {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		//将所有的配置选项最终写入到options实例之中,构建为应用可用的应用配置
		if err := viper.Unmarshal(a.options); err != nil {
			return err
		}
	}

	if !a.silence { //非安静模式,打印一些冗余信息
		fmt.Printf("%v Config file used: `%s`", progressMessage, viper.ConfigFileUsed())
		fmt.Printf("%v Starting %s ...", progressMessage, a.name)
	}

	if a.options != nil { //处理应用的配置,如补全缺失配置等
		if err := a.applyOptionRules(); err != nil {
			return err
		}
	}

	//运行程序(此时a中已携带完整的配置选项,供程序正常运行)
	if a.runFunc != nil {
		return a.runFunc(a.basename)
	}
	return nil

}

func (a *App) applyOptionRules() error {
	//能补齐,则补齐
	if completeableOptions, ok := a.options.(CompleteableOptions); ok {
		if err := completeableOptions.Complete(); err != nil {
			return err
		}
	}
	//验证各个字段的参数
	if errs := a.options.Validate(); len(errs) != 0 {
		return fmt.Errorf("validate options err:%v", errs)
	}
	//打印
	if printableOpt, ok := a.options.(PrintableOptions); ok {
		fmt.Printf("%v Config: `%s`", progressMessage, printableOpt.String())
	}
	return nil

}

func formatBasename(basename string) string {

	if runtime.GOOS == "windows" { //如果是windows 去除.exe后缀
		basename = strings.ToLower(basename)
		basename = strings.TrimSuffix(basename, ".exe")
	}
	return basename
}

func printWorkingDir() {
	wd, _ := os.Getwd()
	fmt.Printf("%v workingDir is: %s", progressMessage, wd)
}

//实现帮助页面的格式化打印
func addCmdTemplate(cmd *cobra.Command, sets NamedFlagSets) {
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout()) //获取终端的宽度,根据宽度来打印
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprint(cmd.OutOrStdout(), usageFmt, cmd.UseLine())
		PrintSections(cmd.OutOrStdout(), sets, cols) //按模块打印
		return nil
	})

	//helpFunc会打印程序的介绍后再打印Usage
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		PrintSections(cmd.OutOrStdout(), sets, cols) //按模块打印
	})

}
