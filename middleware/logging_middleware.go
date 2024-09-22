package middleware

import (
	"bufio"
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type customWriter struct {
	originalWriter gin.ResponseWriter
	responseBody   *bytes.Buffer
	statusCode     int
}

func Logging() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//считываем тело запроса
		requestBody, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			slog.Error("failed to read request body", "error", err)
		}

		//todo разобрать как это работает
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		cWriter := &customWriter{
			originalWriter: ctx.Writer,
			responseBody:   bytes.NewBufferString(""),
		}

		//передаем в контекст gin свой writer, чтобы перехватить тело ответа
		ctx.Writer = cWriter

		//обрабатываем оригинальный запрос и замеряем время
		startHandlingTime := time.Now()
		ctx.Next()
		handlingTime := time.Since(startHandlingTime)

		//логируем
		slog.Debug(
			"incoming request",
			"Method", ctx.Request.Method,
			"host", ctx.Request.Host,
			"path", ctx.Request.URL.Path,
			"client_IP", ctx.ClientIP(),
			"request_handling_time_sec", int(handlingTime)/1000,
			"request_body", string(requestBody),
			"response_body", cWriter.responseBody.String(),
			"status_code", cWriter.statusCode,
		)
	}
}

func (writer *customWriter) Write(b []byte) (int, error) {
	writer.responseBody.Write(b)
	return writer.originalWriter.Write(b)
}

func (writer *customWriter) WriteHeader(statusCode int) {
	writer.statusCode = statusCode
	writer.originalWriter.WriteHeader(statusCode)
}

func (writer *customWriter) Header() http.Header {
	return writer.originalWriter.Header()
}

func (writer *customWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return writer.originalWriter.Hijack()
}

func (writer *customWriter) Flush() {
	writer.originalWriter.Flush()
}

func (writer *customWriter) CloseNotify() <-chan bool {
	return writer.originalWriter.CloseNotify()
}

func (writer *customWriter) Status() int {
	return writer.originalWriter.Status()
}

func (writer *customWriter) Size() int {
	return writer.originalWriter.Size()
}

func (writer *customWriter) WriteString(s string) (int, error) {
	return writer.originalWriter.WriteString(s)
}

func (writer *customWriter) Written() bool {
	return writer.originalWriter.Written()
}

func (writer *customWriter) WriteHeaderNow() {
	writer.originalWriter.WriteHeaderNow()
}

func (writer *customWriter) Pusher() http.Pusher {
	return writer.originalWriter.Pusher()
}
