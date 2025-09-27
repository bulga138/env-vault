package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/DeRuina/timberjack"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Colors for terminal output (ANSI escape codes)
const (
	colorBlue   = "\033[34m" // DEBUG level
	colorCyan   = "\033[36m" // INFO level
	colorGreen  = "\033[32m" // SUCCESS level
	colorRed    = "\033[31m" // ERROR level
	colorReset  = "\033[0m"  // Reset color
	colorYellow = "\033[33m" // WARNING level
)

var outputStr = strings.ToUpper(os.Getenv("LOG_OUTPUT"))
var logFile string
var levelStr = strings.ToUpper(os.Getenv("LOG_LEVEL"))
var logger *zap.Logger
var sugarLogger *zap.SugaredLogger

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: No .env file found or error loading it.")
	}

	outputStr = strings.ToUpper(os.Getenv("LOG_OUTPUT"))
	levelStr = strings.ToUpper(os.Getenv("LOG_LEVEL"))

	logFile = os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "env-vault.log"
	}
}

func Debug(format string, a ...any) {
	if sugarLogger == nil {
		msg := fmt.Sprintf(format, a...)
		fmt.Printf("%s[DEBUG]%s %s\n", colorBlue, colorReset, msg)
		return
	}
	sugarLogger.Debugf(format, a...)
}

func Info(format string, a ...any) {
	if sugarLogger == nil {
		msg := fmt.Sprintf(format, a...)
		fmt.Printf("%s[INFO]%s %s\n", colorCyan, colorReset, msg)
		return
	}
	sugarLogger.Infof(format, a...)
}

func Warn(format string, a ...any) {
	if sugarLogger == nil {
		msg := fmt.Sprintf(format, a...)
		fmt.Printf("%s[WARN]%s %s\n", colorYellow, colorReset, msg)
		return
	}
	sugarLogger.Warnf(format, a...)
}

func Error(format string, a ...any) error {
	if sugarLogger == nil {
		msg := fmt.Sprintf(format, a[:len(a)-1]...)
		err, ok := a[len(a)-1].(error)
		if ok {
			msg = fmt.Sprintf("%s: %v", msg, err)
			fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", colorRed, colorReset, msg)
			return fmt.Errorf("%s", msg)
		}
		fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", colorRed, colorReset, msg)
		return fmt.Errorf("%s", msg)
	}
	sugarLogger.Errorf(format, a...)
	return fmt.Errorf(format, a...)
}

func Fatal(format string, a ...any) {
	if sugarLogger == nil {
		msg := fmt.Sprintf(format, a...)
		fmt.Fprintf(os.Stderr, "%s[FATAL]%s %s\n", colorRed, colorReset, msg)
		os.Exit(1)
	}
	sugarLogger.Fatalf(format, a...)
}

func Success(format string, a ...any) {
	if sugarLogger == nil {
		msg := fmt.Sprintf(format, a...)
		fmt.Printf("%s[SUCCESS]%s %s\n", colorGreen, colorReset, msg)
		return
	}
	sugarLogger.Infof(format, a...)
}

func initLogging() {
	logLevel := getLogLevelFromEnv()
	outputPaths := getLogOutputFromEnv()

	var err error
	logger, err = buildLogger(logLevel, outputPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	sugarLogger = logger.Sugar()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdownChan
		Warn("Received signal %s, shutting down...", sig.String())
		_ = sugarLogger.Sync()
		os.Exit(0)
	}()

	Info("Logging initialized: Level=%s, Output=%s, LogFile=%s",
		strings.ToUpper(logLevel.String()),
		strings.ToUpper(outputStr),
		logFile,
	)
}

func getLogLevelFromEnv() zapcore.Level {
	switch levelStr {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN", "WARNING":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	case "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func getLogOutputFromEnv() []string {
	switch outputStr {
	case "TERMINAL":
		return []string{"stdout"}
	case "FILE":
		return []string{logFile}
	case "BOTH":
		return []string{"stdout", logFile}
	default:
		return []string{"stdout"}
	}
}

func buildLogger(logLevel zapcore.Level, outputPaths []string) (*zap.Logger, error) {
	var cores []zapcore.Core
	loggerOptions := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(2),
	}

	for _, path := range outputPaths {
		var writer zapcore.WriteSyncer
		var encoder zapcore.Encoder

		if path == "stdout" {
			writer = zapcore.Lock(os.Stdout)
			encoderCfg := zapcore.EncoderConfig{
				MessageKey:     "message",
				LevelKey:       "level",
				EncodeLevel:    customLevelEncoder, // Use custom level encoder for colors
				TimeKey:        "",                 // Omit timestamp for cleaner terminal output
				NameKey:        "",                 // Omit logger name
				CallerKey:      "",                 // Omit caller
				StacktraceKey:  "",
				EncodeDuration: zapcore.StringDurationEncoder,
			}
			encoder = zapcore.NewConsoleEncoder(encoderCfg)
		} else {
			if logLevel == zapcore.DebugLevel {
				fmt.Printf("%s[DEBUG]%s  Attempting to configure log file at: %s\n", colorBlue, colorReset, path)
			}
			timberjackLogger := &timberjack.Logger{
				Filename:   path,
				MaxSize:    10,   // megabytes after which new file is created
				MaxBackups: 3,    // number of old files to retain
				MaxAge:     28,   // days
				Compress:   true, // compress/gzip old files
			}

			writer = zapcore.AddSync(timberjackLogger)
			encoderCfg := zap.NewProductionEncoderConfig()
			encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
			encoderCfg.CallerKey = "caller"
			encoderCfg.StacktraceKey = "stacktrace"
			encoder = zapcore.NewJSONEncoder(encoderCfg)
		}
		core := zapcore.NewCore(encoder, writer, logLevel)
		cores = append(cores, core)
	}
	return zap.New(zapcore.NewTee(cores...), loggerOptions...), nil
}

func customLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	var level string
	switch l {
	case zapcore.DebugLevel:
		color = colorGreen
		level = "DEBUG"
	case zapcore.InfoLevel:
		color = colorCyan
		level = "INFO"
	case zapcore.WarnLevel:
		color = colorYellow
		level = "WARN"
	case zapcore.ErrorLevel:
		color = colorRed
		level = "ERROR"
	case zapcore.FatalLevel:
		color = colorRed
		level = "FATAL"
	default:
		color = colorGreen
		level = "SUCCESS"
	}
	enc.AppendString(color + "[" + level + "]" + colorReset)
}
