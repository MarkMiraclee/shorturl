package logger

import (
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"shorturl/internal/config"
)

var Logger *zap.Logger

func InitializeLogger(cfg *config.Config) {
	zapConfig := zap.NewProductionConfig()

	logLevelStr := strings.ToLower(cfg.LogLevel)
	var level zap.AtomicLevel

	switch logLevelStr {
	case "debug":
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		level = zap.NewAtomicLevelAt(zap.InfoLevel) // По умолчанию info
	}

	zapConfig.Level = level

	// Настройка формата логгера
	if strings.ToLower(cfg.LogFormat) == "text" {
		zapConfig.Encoding = "console"
	} else { // По умолчанию json
		zapConfig.Encoding = "json"
	}

	logger, err := zapConfig.Build()
	if err != nil {
		tempLogger, _ := zap.NewProduction()
		tempLogger.Fatal("failed to initialize zap logger", zap.Error(err))
		os.Exit(1)
		return
	}
	Logger = logger
}

// Middleware логирует информацию о запросах и ответах.
func Middleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			uri := r.RequestURI
			method := r.Method

			ww := &ResponseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			statusCode := ww.statusCode
			responseSize := ww.written

			logger.Info("Request processed",
				zap.String("uri", uri),
				zap.String("method", method),
				zap.Duration("duration", duration),
				zap.Int("status_code", statusCode),
				zap.Int("response_size", responseSize),
			)
		})
	}
}

// ResponseWriterWrapper оборачивает http.ResponseWriter для получения кода статуса и размера ответа.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func (w *ResponseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriterWrapper) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += n
	return n, err
}
