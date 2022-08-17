package app

import (
	"bytes"
	goflag "flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"io"
	"strings"
)

// WordSepNormalizeFunc 设置选项的转换,将下划线全部转换为-
func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
	}
	return pflag.NormalizedName(name)
}

func InitFlags(flags *pflag.FlagSet) {
	flags.SetNormalizeFunc(WordSepNormalizeFunc) //设置转换
	flags.AddGoFlagSet(goflag.CommandLine)       //兼容标准库的flag
}

func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		fmt.Printf("%s --%s=%q", color.YellowString("FLAG:"), flag.Name, flag.Value)
	})
}

// NamedFlagSets 定义分组后的的flag
type NamedFlagSets struct {
	Order    []string                  //用于维护顺序,弥补map无序的不足
	FlagSets map[string]*pflag.FlagSet //命令行参数的分组,组名为key,按功能分组
}

// FlagSet 注册一个新的flag分组,如已有对应的分组,则返回该flagset
func (nfs *NamedFlagSets) FlagSet(name string) *pflag.FlagSet {
	if nfs.FlagSets == nil {
		nfs.FlagSets = make(map[string]*pflag.FlagSet)
	}
	if _, ok := nfs.FlagSets[name]; !ok {
		//如果不存在 新建
		nfs.FlagSets[name] = pflag.NewFlagSet(name, pflag.ExitOnError)
		nfs.Order = append(nfs.Order, name)
	}
	return nfs.FlagSets[name]
}

// AddGlobalHelpFlags 添加全局的帮助标志
func AddGlobalHelpFlags(fs *pflag.FlagSet, name string) {
	fs.BoolP("help", "h", false, fmt.Sprintf("help for %s", name))
}

func PrintSections(w io.Writer, fss NamedFlagSets, cols int) {
	//根据选项列表的顺序获取
	//TODO:按模块打印flag的重点实现
	for _, name := range fss.Order {
		fs := fss.FlagSets[name]
		if !fs.HasFlags() { //没有flag则跳过打印
			continue
		}

		wideFS := pflag.NewFlagSet("", pflag.ExitOnError)
		wideFS.AddFlagSet(fs)

		var zzz string
		if cols > 24 {
			zzz = strings.Repeat("z", cols-24)               //刚好让其占满一行,便于删除
			wideFS.Int(zzz, 0, strings.Repeat("z", cols-24)) // zzz一定会被排在每个命令打印的最后,作为标志点
		}

		var buf bytes.Buffer
		//将板块名首字母大写,打印其中包裹的flags;如 Mysql flags:
		fmt.Fprintf(&buf, "\n%s flags:\n\n%s",
			strings.ToUpper(name[:1])+name[1:], wideFS.FlagUsagesWrapped(cols)) //此方法会将每个flag按照列宽度对其打印

		if cols > 24 { //删除最末尾的 --zzzzzzzzzz
			i := strings.Index(buf.String(), zzz)
			lines := strings.Split(buf.String()[:i], "\n")
			fmt.Fprintf(w, strings.Join(lines[:len(lines)-1], "\n")) //删除最后一排的 --zzzzzzz
			fmt.Fprintln(w)
		} else {
			fmt.Fprintf(w, buf.String())
		}

	}

}
