package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

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
