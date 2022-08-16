# Collections of reuseful code

## app

适用性广泛的应用框架,能够提供松散耦合的应用构建方式,提供命令行和配置文件的处理,基于`cobra``viper``pflag`进行构建

## errors

基于`pkg/errors`实现的轻量级错误包,拓展支持JSON输出(利于记录日志),通过Coder接口支持业务错误码和http状态码的映射

## db

常见数据库实例的连接等

## log

基于`zap`二次开发

## shutdown

一个适用性广泛的优雅退出实现


## TODO

- [x] `errors`包的实现
- [ ] `errors`包添加单元测试
- [ ] `log`包的实现
- [ ] `shutdown`包的实现
- [ ] `app`包的实现