package utils

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a sugared zap logger. Initialize with Init or call the helpers below.
var Logger *zap.SugaredLogger

// Init initializes the global logger. If level is empty, it will consult
// LOG_LEVEL env var, then ENV (dev => debug, otherwise info).
func Init(level string) error {
	if level == "" {
		level = os.Getenv("LOG_LEVEL")
		if level == "" {
			if os.Getenv("ENV") == "dev" {
				level = "debug"
			} else {
				level = "info"
			}
		}
	}

	var cfg zap.Config
	if strings.ToLower(level) == "debug" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		// Production: use JSON by default
		cfg.Encoding = "json"
	}

	// Map level strings to zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn", "warning":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	Logger = logger.Sugar()
	return nil
}

// Sync flushes any buffered logs. Safe to call even if logger is nil.
func Sync() {
	if Logger == nil {
		return
	}
	_ = Logger.Sync()
}

// Convenience wrappers that fall back to fmt.Printf if Logger isn't initialized.
func Debugf(format string, args ...interface{}) {
	if Logger == nil {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
		return
	}
	Logger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	if Logger == nil {
		fmt.Printf("[INFO] "+format+"\n", args...)
		return
	}
	Logger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	if Logger == nil {
		fmt.Printf("[WARN] "+format+"\n", args...)
		return
	}
	Logger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	if Logger == nil {
		fmt.Printf("[ERROR] "+format+"\n", args...)
		return
	}
	Logger.Errorf(format, args...)
}

// Structured helpers that accept key/value pairs (fall back if Logger is nil)
func Infow(msg string, keysAndValues ...interface{}) {
	if Logger == nil {
		// fallback to simple print
		fmt.Printf("[INFO] %s %v\n", msg, keysAndValues)
		return
	}
	Logger.Infow(msg, keysAndValues...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	if Logger == nil {
		fmt.Printf("[DEBUG] %s %v\n", msg, keysAndValues)
		return
	}
	Logger.Debugw(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	if Logger == nil {
		fmt.Printf("[WARN] %s %v\n", msg, keysAndValues)
		return
	}
	Logger.Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	if Logger == nil {
		fmt.Printf("[ERROR] %s %v\n", msg, keysAndValues)
		return
	}
	Logger.Errorw(msg, keysAndValues...)
}
