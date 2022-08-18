package app

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"os"
)

// Command 代表一个命令行程序中的子命令,每个命令有自己的命令行选项和RunFunc
type Command struct {
	usage    string
	desc     string
	options  CliOptions
	commands []*Command //代表着此命令旗下的子命令
	runFunc  RunCommandFunc
}

// CommandOption 选项模式创建一个子命令
type CommandOption func(*Command)

// WithCommandOptions 设置该命令的命令行选项
func WithCommandOptions(opt CliOptions) CommandOption {
	return func(c *Command) {
		c.options = opt
	}
}

type RunCommandFunc func(args []string) error

func WithCommandRunFunc(run RunCommandFunc) CommandOption {
	return func(c *Command) {
		c.runFunc = run
	}
}

func NewCommand(usage string, desc string, opts ...CommandOption) *Command {
	c := &Command{
		usage: usage,
		desc:  desc,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c

}

// AddCommand 向c中添加一个 子命令
func (c *Command) AddCommand(cmd *Command) {
	c.commands = append(c.commands, cmd)
}

func (c *Command) AddCommands(cmd ...*Command) {
	c.commands = append(c.commands, cmd...)
}

//将Command转换为cobra.Command
//命令构建的步骤:
//1. 构建cobra.Command基础
//2. 递归处理子命令,附加到cobra上
//3. 设置runFunc执行的函数
//4. 将options提供的flag分组加入到cmd
//5. 添加help flag
func (c *Command) cobraCommand() *cobra.Command {
	cmd := &cobra.Command{ //先构建一个根
		Use:   c.usage,
		Short: c.desc,
	}
	cmd.SetOut(os.Stdout)
	cmd.Flags().SortFlags = true
	//如果这个命令有子命令的话,递归的将所有的子命令集中
	if len(c.commands) > 0 {
		for _, command := range c.commands {
			cmd.AddCommand(command.cobraCommand())
		}
	}

	if c.runFunc != nil {
		cmd.Run = c.runCommand
	}

	if c.options != nil {
		for _, f := range c.options.Flags().FlagSets {
			cmd.Flags().AddFlagSet(f)
		}
	}
	addHelpCommandFlag(c.usage, cmd.Flags()) //当前命令添加help flag
	return cmd
}

func (c *Command) runCommand(cmd *cobra.Command, args []string) {
	if c.runFunc != nil {
		if err := c.runFunc(args); err != nil {
			fmt.Printf("%v %v\n", color.RedString("Error:"), err)
			os.Exit(1)
		}
	}

}
