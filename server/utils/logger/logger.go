package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	PANIC
)

var levelNames = []string{
	"TRACE",
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"FATAL",
	"PANIC",
}

// Logger 自定义日志器
type Logger struct {
	level  LogLevel
	writer io.Writer
	mu     sync.Mutex
	prefix string
	colors bool
}

func isFile(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	// 检查是否为标准输入/输出/错误
	if file == os.Stdin || file == os.Stdout || file == os.Stderr {
		return false
	}

	// 其他情况视为真正的文件
	return true
}

func WriteLog(w io.Writer, line string) {
	if isFile(w) {
		w.Write([]byte(line))
	} else {
		fmt.Println(line)
	}
}

// NewLogger 创建新日志器
func newLogger(level LogLevel, writer io.Writer) *Logger {
	// Disable colors by default on Windows since cmd.exe doesn't support ANSI color codes
	colorsEnabled := false
	return &Logger{
		level:  level,
		writer: writer,
		colors: colorsEnabled,
	}
}

// NewFileLogger 创建文件日志器
func newFileLogger(level LogLevel, filePath string) (*Logger, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return newLogger(level, file), nil
}

// SetPrefix 设置日志前缀
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// SetColors 启用/禁用颜色
func (l *Logger) SetColors(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.colors = enabled
}

// getCaller 获取调用 Infof/Debugf 等函数的源码位置
func (l *Logger) getCaller() string {
	pc, file, line, ok := runtime.Caller(3)
	if !ok {
		return "???:0"
	}
	_, filename := filepath.Split(file)
	fn := runtime.FuncForPC(pc)
	funcName := fn.Name()
	parts := strings.Split(funcName, ".")
	return fmt.Sprintf("%s:%s:%d", filename, parts[len(parts)-1], line)
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]

	var colorStart, colorEnd string
	if l.colors {
		switch level {
		case TRACE:
			colorStart = "\033[34m" // Blue
		case DEBUG:
			colorStart = "\033[36m" // Cyan
		case INFO:
			colorStart = "\033[32m" // Green
		case WARN:
			colorStart = "\033[33m" // Yellow
		case ERROR:
			colorStart = "\033[31m" // Red
		case FATAL:
			colorStart = "\033[35m" // Magenta
		case PANIC:
			colorStart = "\033[31m" // Red
		}
		colorEnd = "\033[0m"
	}

	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s [%s] %s%s\n", now, levelName, l.prefix, message)

	if l.colors {
		logLine = colorStart + logLine + colorEnd
	}

	WriteLog(l.writer, logLine)

	if level == PANIC {
		// Use panic instead of os.Exit so defer statements can execute
		panic("fatal error")
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// 各种日志级别的方法
func (l *Logger) Tracef(format string, args ...interface{}) {
	if TRACE < l.level {
		return
	}
	caller := l.getCaller()
	l.log(TRACE, "%s "+format, append([]any{caller}, args...)...)
}
func (l *Logger) Debugf(format string, args ...interface{}) {
	if DEBUG < l.level {
		return
	}
	caller := l.getCaller()
	l.log(DEBUG, "%s "+format, append([]any{caller}, args...)...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	if INFO < l.level {
		return
	}
	caller := l.getCaller()
	l.log(INFO, "%s "+format, append([]any{caller}, args...)...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	if WARN < l.level {
		return
	}
	caller := l.getCaller()
	l.log(WARN, "%s "+format, append([]any{caller}, args...)...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	if ERROR < l.level {
		return
	}
	caller := l.getCaller()
	l.log(ERROR, "%s "+format, append([]any{caller}, args...)...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	if FATAL < l.level {
		return
	}
	caller := l.getCaller()
	l.log(FATAL, "%s "+format, append([]any{caller}, args...)...)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	if PANIC < l.level {
		return
	}
	caller := l.getCaller()
	l.log(PANIC, "%s "+format, append([]any{caller}, args...)...)
}

func (l *Logger) Trace(args ...interface{}) {
	if TRACE < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(TRACE, "%s", msg)
}

func (l *Logger) Debug(args ...interface{}) {
	if DEBUG < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(DEBUG, "%s", msg)
}

func (l *Logger) Info(args ...interface{}) {
	if INFO < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(INFO, "%s", msg)
}

func (l *Logger) Warn(args ...interface{}) {
	if WARN < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(WARN, "%s", msg)
}

func (l *Logger) Error(args ...interface{}) {
	if ERROR < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(ERROR, "%s", msg)
}

func (l *Logger) Fatal(args ...interface{}) {
	if FATAL < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(FATAL, "%s", msg)
}
func (l *Logger) Panic(args ...interface{}) {
	if PANIC < l.level {
		return
	}
	caller := l.getCaller()
	msg := fmt.Sprint(append([]interface{}{caller}, args...)...)
	l.log(PANIC, "%s", msg)
}

// 全局日志器
var logger *Logger

func init() {
	logger = newLogger(INFO, os.Stdout)
	// logger, _ = newFileLogger(INFO, "log.log")
}

// SetGlobalLogger 设置全局日志器
func SetLogger(l *Logger) {
	logger = l
}

func GetLogger() *Logger {
	return logger
}

// 全局日志方法

func SetPrefix(prefix string) {
	logger.SetPrefix(prefix)
}

func SetLevel(level LogLevel) {
	logger.SetLevel(level)
}

func Trace(args ...interface{}) {
	logger.Trace(args...)
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Panic(args ...interface{}) {
	logger.Panic(args...)
}

func Debugf(format string, args ...any) {
	logger.Debugf(format, args...)
}

func Infof(format string, args ...any) {
	logger.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	logger.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	logger.Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	logger.Fatalf(format, args...)
}

func Panicf(format string, args ...any) {
	logger.Panicf(format, args...)
}
