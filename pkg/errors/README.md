# errors 实现重点


## 需求

- 能够记录堆栈
- 支持业务错误码的映射
- 支持Wrap/Unwrap方法
- 支持格式化打印,包括JSON
- 支持多种打印方式,如`%v` `%+v`之类的形式

## 实现

### 如何实现记录堆栈的?

[如何在Go函数中得到调用者的函数名?](https://colobu.com/2018/11/03/get-function-name-in-go/)

由`errors`包的`New`方法创建出来的错误`baseErr`,具体有一个`*stack`字段
```go
type baseError struct {
	msg string
	*stack
}
```
该字段用于记录调用堆栈的信息,其本质是一个`[]uintptr`,`uintptr`是一个无符号的整型数，足以保存一个地址,代表的就是整个调用的堆栈
```go
//用于记录堆栈深度
type stack []uintptr //代表调用堆栈,由许多个帧组成

func callers() *stack {
	var pcs [stackDepth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
```
`runtime.Callers(3, pcs[:])` 3的含义: callers自身(1),callers这个函数(2),调用callers的函数3(如withstack),跳过记录这3个函数;调用此函数将会记录当前用户调用处的函数栈程序计数器的情况(一个[]uintptr的切片)

获取到了程序计数器(pc)后,要对其进行格式化打印,中间转换成了Frame类型(uintptr别名)
```go
type Frame uintptr //代表堆栈中的一个帧,他的值是程序计数器+1

func (f Frame) pc() uintptr { return uintptr(f) - 1 }

func (f Frame) file() string { //根据给定的帧,返回当前程序计数器所在的函数的所在文件名
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}
func (f Frame) line() int { //根据给定的帧,返回当前程序计数器所在的函数函数所在行号
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

func (f Frame) name() string { //返回当前计数器所在的函数名称
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}
```

其值其实就是pc的值减去1(这样能够在打印时减少最新的函数的重复打印)

### 如何实现格式化打印的?

最关键的就是实现`fmt.Formatter`接口,考虑到不同错误的层层嵌套的问题,本包中每一种错误都实现了该接口,如:
```go
func (f *baseError) Format(s fmt.State, verb rune) {
	switch verb {****
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, f.msg)
			f.stack.Format(s, verb) //*stack也实现了Formatter接口
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, f.msg)
	case 'q':
		fmt.Fprintf(s, "%q", f.msg)
	}
}
```
接口定义:
```go
type State interface {
	// Write is the function to call to emit formatted output to be printed.
	Write(b []byte) (n int, err error)
	// Width returns the value of the width option and whether it has been set.
	Width() (wid int, ok bool)
	// Precision returns the value of the precision option and whether it has been set.
	Precision() (prec int, ok bool)

	// Flag reports whether the flag c, a character, has been set.
	Flag(c int) bool
}
```
`verb`代表的是打印的类型,如`v``s``q`等,可自定义,`fmt.State`是一个状态接口,其提供Writer的实现,能够将内容输出至writer  
`Flag`代表格式化提供的flag如`%+v`中的`+`,这样就能够通过`switch`语句实现不同输出格式输出不同的内容


其中最重要的格式化方法就是`withCode`类型的方法,`%#v`会以JSON格式输出最顶层的错误,`%+v`会将整条调用链上的错误都打印出来


### 如何适配业务错误码和http码?

定义了一个Coder接口
```go
//Coder 接口定义了一个错误码的详细信息接口
type Coder interface {
	HTTPStatus() int//对应的http码
	String() string//对用户安全的错误信息
	Reference() string //该错误的参考信息
	Code() int//业务错误码
}
```
并在包内维护了一个key为业务错误码,value为Coder的map
```go
var (
	unknownCoder = defaultCoder{C: 1, HTTP: http.StatusInternalServerError,
		Ext: "An internal server error occurred", Ref: "none"}
	codes   = map[int]Coder{}
	codeMux = &sync.Mutex{}
)
```
各个项目可以实现自己的Coder实例,创建具有对应http码,业务错误码,对用户安全的错误信息,之后通过通过`Register`或`MustRegister`将Coder注册到map中维护
```go
func Register(coder Coder) {
	if coder.Code() == 0 {
		panic(`请设置非0错误码`)
	}
	codeMux.Lock()
	defer codeMux.Unlock()
	codes[coder.Code()] = coder
}
//也提供了查询相关的方法

func ParseCoder(err error) Coder {
	if err == nil {
		return nil
	}

	if v, ok := err.(*withCode); ok { //必须是*withCode类型
		if coder, ok := codes[v.code]; ok {
			return coder
		}
	}
	return unknownCoder
}

```
可以通过Coder返回具有业务码,参考信息,http状态码,安全错误信息的响应给到前端,后系统内部可以获得错误的详细信息,用于排障分析(这部分信息不便用户知道)





### 一些技巧

#### 使用`bytes.Buffer`高效构建输出内容

withCode的Format方法中创建了`str := bytes.NewBuffer([]byte{})`并将其引用传递,其格式化的数据根据处理逻辑一层一层的写入到Buffer中,如:  
`fmt.Fprintf(str, "%s%s - #%d %s", sep, finfo.err, k, finfo.message)` 这样在处理嵌套err的情况时非常好用,每一层err将信息写入到buffer中即可  

最终要输出时也会非常方便,如输出为Json
```go
	if modeJSON {
			var byts []byte
			byts, _ = json.Marshal(jsonData)
			str.Write(byts)
		}
```
或者输出为字符串:
```go
fmt.Fprintf(state, "%s", strings.Trim(str.String(), "\r\n\t"))
```

#### 善于使用标志变量
如
```go
var (
			flagDetail, flagTrace, modeJSON bool
		)
```
在后面函数中有利于条件的灵活判断















