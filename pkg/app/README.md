# App包

## 重点需求

- 支持组建子命令
- 能够格式化打印help页面(flag要按模块分组显示)
- flag的分组管理,如global组,mysql组,redis组等
- flag集有层次(如每个命令只获取自己的flag或全局flag)
- 支持命令行选项和配置文件的统一,且命令行选项优先级高于配置文件的相同名称
- 支持环境变量获取

## 如何实现

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