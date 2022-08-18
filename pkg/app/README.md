# App包

## 重点需求

- 支持组建子命令
- 能够格式化打印help页面(flag要按模块分组显示)
- flag的分组管理,如global组,mysql组,redis组等
- flag集有层次(如每个命令只获取自己的flag或全局flag)
- 支持命令行选项和配置文件的统一,且命令行选项优先级高于配置文件的相同名称
- 支持环境变量获取

## **如何实现**

### App

App是一个应用程序的抽象类,具备二进制文件名称(basename),非选项参数(args),子命令([]*Command),命令行框架(cmd)  
以及其命令行选项(CliOptions),App通过选项模式提供给外界创建方法,`NewApp`就是整个构建应用的过程

### 命令行框架的构建

App的`buildCommand`方法中,完成了cmd的构建(命令行框架),主要有以下步骤

1. 创建根命令`cmd`
```go
	//1.构建根命令
	cmd := cobra.Command{
		Use:           formatBasename(a.basename),
		Short:         a.name,
		Long:          a.description,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          a.args,
	}
```

2. 查看是否设置有子命令,如有,遍历附加到根命令上

```go
	if len(a.commands) > 0 {
		for _, command := range a.commands {
			cmd.AddCommand(command.cobraCommand())
		}
		//如果有子命令的话,增加额外一个help命令,以查看子命令列表
		cmd.SetHelpCommand(helpCommand(formatBasename(a.basename)))
	}
```

3. 注册程序的入口函数`RunE`

```go
	if a.runFunc != nil {
		cmd.RunE = a.runCommand
	}
```

在`runCommand`中会在cmd.Execute被执行,会先处理一些配置相关的合并之后,再运行用户传入的回调函数,即 `a.RunFunc`

4. 将传入的`CliOption`使用其`Flags()`方法将其所有的选项按分组分类成flagSet,绑定到cmd上,并根据需求设置全局的flag,如version,config等

```go
var namedFlagSets NamedFlagSets
	if a.options != nil {
		namedFlagSets = a.options.Flags()
		for _, set := range namedFlagSets.FlagSets {
			cmd.Flags().AddFlagSet(set) //将分组的flag绑定到主命令中
		}
	}
```

5. 修改cmd的默认帮助信息页面`	addCmdTemplate(&cmd, namedFlagSets) `

```go
func addCmdTemplate(cmd *cobra.Command, namedFlagSets cliflag.NamedFlagSets) {
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)

		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine()) //先打印cmd.Long,然后Usage
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})
}
```
6. 将构建好的cmd返回,后客户端方面调用`Execute`,将会最终执行用户传入的runFunc,并在之前处理好命令行参数和配置文件的参数,合并为最终的应用参数,应用可以此正常启动





