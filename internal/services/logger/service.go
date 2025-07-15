package logger

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type RoundTripper struct {
	Logger *zap.Logger
	Proxy  http.RoundTripper
}

func NewRoundTripper(logger *zap.Logger) *RoundTripper {
	return &RoundTripper{
		Logger: logger,
		Proxy:  http.DefaultTransport,
	}
}

func (l *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := l.Proxy.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		l.Logger.Error("HTTP request failed",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Logger.Error("Failed to read response body",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return resp, err
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	l.Logger.Info("HTTP request completed",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.ByteString("body_snipped", bodyBytes),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration),
	)

	return resp, nil
}
