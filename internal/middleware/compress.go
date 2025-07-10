package middleware

import (
	"compress/gzip"
	"go.uber.org/zap"
	"io"
	"net/http"
	"shorturl/internal/logger"
	"strings"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzipResponseWriter) Flush() {
	if f, ok := w.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

// GzipResponse middleware сжимает ответы
func GzipResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer func() {
			if err := gz.Close(); err != nil {
				logger.Logger.Error("Error closing gzip writer", zap.Error(err)) // Используем zap.Error
			}
		}()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // Content-Length нужно удалить, т.к. сжатый размер отличается

		next.ServeHTTP(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}

// GzipRequest middleware распаковывает тело запроса
func GzipRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			r.Body = gz
			defer func() {
				if err := gz.Close(); err != nil {
					logger.Logger.Error("Error closing gzip reader", zap.Error(err)) // Используем zap.Error
				}
			}()
		}
		next.ServeHTTP(w, r)
	})
}
