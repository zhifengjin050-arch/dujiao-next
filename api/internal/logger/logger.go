package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultLogDirName    = "logs"
	defaultLogFilename   = "app.log"
	defaultLogMaxSizeMB  = 100
	defaultLogMaxBackups = 7
	defaultLogMaxAgeDays = 30
	defaultLogCompress   = true
)

// Options ??????
type Options struct {
	Dir        string
	Filename   string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// L ?????????
var L *zap.Logger

var (
	fallbackOnce sync.Once
	fallbackLog  *zap.Logger
)

// Init ???????
func Init(mode string, options Options) *zap.Logger {
	L = New(mode, options)
	if L == nil {
		L = fallbackLogger()
	}
	zap.ReplaceGlobals(L)
	return L
}

// New ??????
func New(mode string, options Options) *zap.Logger {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if strings.EqualFold(strings.TrimSpace(mode), "debug") {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.MessageKey = "message"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.MillisDurationEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	if strings.EqualFold(strings.TrimSpace(mode), "debug") {
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
		return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	writeSyncer, err := newFileWriteSyncer(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init failed, fallback to stdout: %v\n", err)
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
		return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		level,
	)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

// StdLogger ??????? log ? logger
func StdLogger() *log.Logger {
	return zap.NewStdLog(Z())
}

// Z ????????????
func Z() *zap.Logger {
	if L != nil {
		return L
	}
	return fallbackLogger()
}

// S ????? SugaredLogger
func S() *zap.SugaredLogger {
	return Z().Sugar()
}

// SW ????????? SugaredLogger
func SW(kv ...interface{}) *zap.SugaredLogger {
	if len(kv) == 0 {
		return S()
	}
	return S().With(kv...)
}

// Debugw ?? debug ????
func Debugw(message string, kv ...interface{}) {
	S().Debugw(message, kv...)
}

// Infow ?? info ????
func Infow(message string, kv ...interface{}) {
	S().Infow(message, kv...)
}

// Warnw ?? warn ????
func Warnw(message string, kv ...interface{}) {
	S().Warnw(message, kv...)
}

// Errorw ?? error ????
func Errorw(message string, kv ...interface{}) {
	S().Errorw(message, kv...)
}

func fallbackLogger() *zap.Logger {
	fallbackOnce.Do(func() {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "time"
		encoderConfig.MessageKey = "message"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeDuration = zapcore.MillisDurationEncoder
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			zap.NewAtomicLevelAt(zap.InfoLevel),
		)
		fallbackLog = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	})
	return fallbackLog
}

func newFileWriteSyncer(options Options) (zapcore.WriteSyncer, error) {
	logFilePath, err := resolveLogFilePath(options)
	if err != nil {
		return nil, err
	}

	writer := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    normalizePositiveInt(options.MaxSizeMB, defaultLogMaxSizeMB),
		MaxBackups: normalizePositiveInt(options.MaxBackups, defaultLogMaxBackups),
		MaxAge:     normalizePositiveInt(options.MaxAgeDays, defaultLogMaxAgeDays),
		Compress:   options.Compress,
	}
	if !options.Compress {
		writer.Compress = false
	} else {
		writer.Compress = defaultLogCompress
	}
	return zapcore.AddSync(writer), nil
}

func resolveLogFilePath(options Options) (string, error) {
	dir := strings.TrimSpace(options.Dir)
	if dir == "" {
		workDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve workdir failed: %w", err)
		}
		dir = filepath.Join(workDir, defaultLogDirName)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create log dir failed: %w", err)
	}

	filename := strings.TrimSpace(options.Filename)
	if filename == "" {
		filename = defaultLogFilename
	}

	logFilePath := filepath.Join(dir, filename)
	if err := ensureLogFileWritable(logFilePath); err != nil {
		return "", err
	}

	return logFilePath, nil
}

func ensureLogFileWritable(logFilePath string) error {
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close log file failed: %w", err)
	}
	return nil
}

func normalizePositiveInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
