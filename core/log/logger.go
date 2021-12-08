package log

import (
	"context"
	"fmt"
	rotate "github.com/lestrrat-go/file-rotatelogs"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"strings"
	"time"
)

var (
	_filePathName = map[zapcore.Level]string{
		zapcore.DebugLevel: "debug.log",
		zapcore.InfoLevel:  "info.log",
		zapcore.WarnLevel:  "warn.log",
		zapcore.ErrorLevel: "error.log",
	}
)

var (
	_enableFunc = map[zapcore.Level]zap.LevelEnablerFunc{
		zapcore.DebugLevel: zap.LevelEnablerFunc(func(level zapcore.Level) bool {return level >= zapcore.DebugLevel}),
		zapcore.InfoLevel:  zap.LevelEnablerFunc(func(level zapcore.Level) bool {return level >= zapcore.InfoLevel}),
		zapcore.WarnLevel:  zap.LevelEnablerFunc(func(level zapcore.Level) bool {return level >= zapcore.WarnLevel}),
		zapcore.ErrorLevel: zap.LevelEnablerFunc(func(level zapcore.Level) bool {return level >= zapcore.ErrorLevel}),
	}
)

type Logger struct {
	logger *zap.Logger
}

func InitLogger(path, name, split string, days int) *Logger {
	// split
	var s time.Duration
	var f string
	switch strings.ToLower(split) {
	case "hour":
		s = time.Hour
		f = "-%Y%m%d%H.log"
	case "day":
		s = time.Hour * 24
		f = "-%Y%m%d.log"
	default:
		s = time.Hour * 24
		f = "-%Y%m%d.log"
	}
	// 生成core
	if path[len(path)-1] != '/' {
		path += "/"
	}
	fullName := path + name
	r, err := rotate.New(
		strings.Replace(fullName, ".log", "", -1) + f,
		rotate.WithLinkName(fullName),
		rotate.WithMaxAge(time.Hour*24*time.Duration(days)),
		rotate.WithRotationTime(s),
	)
	if err != nil {
		panic(fmt.Sprintf("init logger new roate_log err:%v", err))
	}
	core := createCore(r, _enableFunc[zapcore.InfoLevel], false, false, "json")
	// 生成logger
	return &Logger{
		logger: zap.New(core, zap.AddCaller()),
	}
}

func InitGroupLogger(path, level ,split string, days int) *Logger {
	// 判断等级
	var zapLevel zapcore.Level
	level = strings.ToLower(level)
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		panic(fmt.Sprintf("init logger,level not support, level:%s", level))
	}
	// split
	var s time.Duration
	var f string
	switch strings.ToLower(split) {
	case "hour":
		s = time.Hour
		f = "-%Y%m%d%H.log"
	case "day":
		s = time.Hour * 24
		f = "-%Y%m%d.log"
	default:
		s = time.Hour * 24
		f = "-%Y%m%d.log"
	}
	// 生成core
	if path[len(path)-1] != '/' {
		path += "/"
	}
	var cores []zapcore.Core
	for fileLevel, fileName := range _filePathName {
		if fileLevel >= zapLevel {
			fullName := path + fileName
			r, err := rotate.New(
				strings.Replace(fullName, ".log", "", -1) + f,
				rotate.WithLinkName(fullName),
				rotate.WithMaxAge(time.Hour*24*time.Duration(days)),
				rotate.WithRotationTime(s),
			)
			if err != nil {
				panic(fmt.Sprintf("init logger new roate_log err:%v", err))
			}
			cores = append(cores, createCore(r, _enableFunc[fileLevel], true, true, "consul"))
		}
	}
	// 生成logger
	core := zapcore.NewTee(cores...)
	return &Logger{
		logger: zap.New(core, zap.AddCaller()),
	}
}

func createCore(r *rotate.RotateLogs, enableFunc zap.LevelEnablerFunc, withCaller bool, withLevel bool, format string) zapcore.Core {
	opts := []zapcore.WriteSyncer{
		zapcore.AddSync(r),
	}
	syncWriter := zapcore.NewMultiWriteSyncer(opts...)
	// 自定义时间输出格式
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	// 自定义日志级别显示
	customLevelEncoder := func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + level.CapitalString() + "]")
	}
	// 自定义文件：行号输出项
	customCallerEncoder := func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		_, caller.File, caller.Line, _ = runtime.Caller(7)
		enc.AppendString("[" + caller.TrimmedPath() + "]")
	}
	// encoder配置
	encoderConf := zapcore.EncoderConfig{
		MessageKey:       "msg",
		TimeKey:          "time",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		ConsoleSeparator: " ",
		EncodeTime:       customTimeEncoder,   // 自定义时间格式
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeName:       zapcore.FullNameEncoder,
	}
	if withCaller {
		encoderConf.CallerKey = "caller" // 打印文件名和行数
		encoderConf.EncodeCaller = customCallerEncoder // 全路径编码器
	}
	if withLevel {
		encoderConf.LevelKey = "level"
		encoderConf.EncodeLevel = customLevelEncoder // 小写编码器
	}
	if format == "json" {
		return zapcore.NewCore(zapcore.NewJSONEncoder(encoderConf),
			syncWriter, enableFunc)
	}
	return zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConf),
		syncWriter, enableFunc)
}

func (l *Logger) trace(ctx context.Context) string {
	var traceID string
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		traceID = ""
	} else {
		if sc, ok := span.Context().(jaeger.SpanContext); ok {
			traceID =  sc.TraceID().String()
		}
	}
	return traceID
}

func (l *Logger) write(ctx context.Context, level zapcore.Level, format string, args ...interface{}) {
	if ce := l.logger.Check(level, fmt.Sprintf(format, args...)); ce != nil {
		trace := l.trace(ctx)
		if trace != "" {
			ce.Write(
				zap.String("trace_id", trace),
			)
		} else {
			ce.Write()
		}
	}
}

func (l *Logger) Debug(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, zapcore.DebugLevel, format, args...)
}

func (l *Logger) Info(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, zapcore.InfoLevel, format, args...)
}

func (l *Logger) Warn(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, zapcore.WarnLevel, format, args...)
}

func (l *Logger) Error(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, zapcore.ErrorLevel, format, args...)
}

// 框架默认日志
var _logger *Logger

func init() {
	_defaultLogger := InitGroupLogger("./log", "debug", "hour", 7)
	SetLogger(_defaultLogger)
}

func SetLogger(l *Logger) {
	_logger = l
}

func Debug(ctx context.Context, format string, args ...interface{}) {
	_logger.Debug(ctx, format, args...)
}

func Info(ctx context.Context, format string, args ...interface{}) {
	_logger.Info(ctx, format, args...)
}

func Warn(ctx context.Context, format string, args ...interface{}) {
	_logger.Warn(ctx, format, args...)
}

func Error(ctx context.Context, format string, args ...interface{}) {
	_logger.Error(ctx, format, args...)
}
