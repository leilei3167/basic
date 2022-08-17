package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

//withCode 的格式化打印实现

type formatInfo struct {
	code    int
	message string
	err     string
	stack   *stack
}

//Format 实现Formatter接口
//%s,%v 打印用户安全的脱敏错误信息
//标志:
//# 将错误信息以json格式打印,便于日志记录
//- 打印调用者的信息
//+ 打印整个调用堆栈

func (w *withCode) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v':
		str := bytes.NewBuffer([]byte{})
		jsonData := []map[string]any{}

		var (
			flagDetail, flagTrace, modeJSON bool
		)
		if state.Flag('#') {
			modeJSON = true
		}
		if state.Flag('-') {
			flagDetail = true
		}
		if state.Flag('+') {
			flagTrace = true
		}

		sep := ""
		errs := list(w) //获取整条错误链上的错误
		length := len(errs)
		for k, e := range errs {
			//将每个错误按统一的格式 格式化
			finfo := buildFormatInfo(e)
			//构建数据(此处至少打印最新的错误信息)
			jsonData, str = format(length-k-1, jsonData, str, finfo, sep, flagDetail, flagTrace, modeJSON)
			sep = ";"

			if !flagTrace { //如果不需要所有堆栈跟踪,打印最新错误即可终止
				break
			}
			if !flagDetail && !flagTrace && !modeJSON {
				break
			}
		}
		//如果需要将json打印,则将累计的所有信息编码
		if modeJSON {
			var byts []byte
			byts, _ = json.Marshal(jsonData)
			str.Write(byts)
		}
		fmt.Fprintf(state, "%s", strings.Trim(str.String(), "\r\n\t"))
	default:
		finfo := buildFormatInfo(w)
		fmt.Fprintf(state, finfo.message)
	}
}

//buildFormatInfo 将一个错误转换为待打印的结构
func buildFormatInfo(e error) *formatInfo {
	var finfo *formatInfo

	switch err := e.(type) { //类型选择,根据不同类型的错误进行判断
	case *baseError:
		finfo = &formatInfo{
			code:    unknownCoder.Code(),
			message: err.msg,
			err:     err.msg,
			stack:   err.stack,
		}
	case *withStack:
		finfo = &formatInfo{
			code:    unknownCoder.Code(),
			message: err.Error(),
			err:     err.Error(),
			stack:   err.stack,
		}
	case *withCode:
		coder, ok := codes[err.code] //从中获取已注册的coder
		if !ok {
			coder = unknownCoder
		}

		extMsg := coder.String() //获取面向用户的安全错误信息
		if extMsg == "" {
			extMsg = err.err.Error()
		}
		finfo = &formatInfo{
			code:    coder.Code(),
			message: extMsg,          //此处是对用户安全的信息(注册错误码时指定的)
			err:     err.err.Error(), //此处是对内错误信息
			stack:   err.stack,
		}

	default:
		finfo = &formatInfo{
			code:    unknownCoder.Code(),
			message: err.Error(),
			err:     err.Error(),
		}
	}
	return finfo
}

func format(k int, jsonData []map[string]any, str *bytes.Buffer, finfo *formatInfo,
	sep string, flagDetail, flagTrace, modeJSON bool) ([]map[string]any, *bytes.Buffer) {
	if modeJSON { //如果以JSON形式输出
		data := map[string]any{}
		if flagDetail || flagTrace { //如果要求打印堆栈
			data = map[string]any{
				"message": finfo.message,
				"code":    finfo.code,
				"error":   finfo.err,
			}
			caller := fmt.Sprintf("#%d", k)
			if finfo.stack != nil {
				f := Frame((*finfo.stack)[0]) //只将最近的调用情况
				caller = fmt.Sprintf("%s %s:%d (%s)", caller, f.file(), f.line(), f.name())
			}
			data["caller"] = caller
		} else { //不需要打印堆栈的话,打印错误信息即可
			data["error"] = finfo.message
		}
		jsonData = append(jsonData, data)
	} else { //不以JSON输出
		if flagDetail || flagTrace {
			if finfo.stack != nil {
				f := Frame((*finfo.stack)[0])
				fmt.Fprintf(str, "%s%s - #%d [%s:%d (%s)](%d) %s",
					sep, finfo.err, k, f.file(), f.line(), f.name(), finfo.code, finfo.message)
			} else { //没有记录堆栈
				fmt.Fprintf(str, "%s%s - #%d %s", sep, finfo.err, k, finfo.message)
			}

		} else { //不需要打印堆栈
			fmt.Fprintf(str, finfo.message)
		}
	}

	return jsonData, str
}

//list 获取整条错误链上的错误
func list(e error) []error {
	ret := []error{}

	if e != nil {
		if w, ok := e.(interface{ Unwrap() error }); ok {
			//如果e具有Unwrap方法,将其加入后,将其所持有的error也加入
			ret = append(ret, e)
			ret = append(ret, list(w.Unwrap())...)
		} else {
			ret = append(ret, e)
		}

	}
	return ret
}
