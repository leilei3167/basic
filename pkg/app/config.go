package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leilei3167/basic/pkg/base/util/homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const configFlagName = "config"

var cfgFile string //要包含拓展名

func init() {
	//向默认set中注册
	pflag.StringVarP(&cfgFile, "config", "c", cfgFile, "指定的配置文件")
}

//向指定的flagset中添加配置文件的选项
func addConfigFlag(basename string, fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(configFlagName)) //从默认set中取出,加入到指定set中

	viper.AutomaticEnv() //自动识别环境变量
	//环境变量的前缀为二进制文件名大写(所有-转换为下划线),如 API_SERVER
	viper.SetEnvPrefix(strings.Replace(strings.ToUpper(basename), "-", "_", -1))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	cobra.OnInitialize(func() { //设置cobra中,在Execute执行前要执行的函数,此处会在runCommand之前被执行
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			//该flag未被指定,则从默认的地址获取配置文件
			viper.AddConfigPath(".") //当前文件夹

			if names := strings.Split(basename, "-"); len(names) > 1 {
				viper.AddConfigPath(filepath.Join(homedir.HomeDir(), "."+names[0])) // /home/lei/.api
				viper.AddConfigPath(filepath.Join("/etc", names[0]))
			}
			viper.SetConfigName(basename) //配置文件名称
		}

		//读取配置文件,失败则直接退出程序
		if err := viper.ReadInConfig(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: failed to read config file(%s):%v\n", cfgFile, err)
			os.Exit(1)
		}
	})

}
