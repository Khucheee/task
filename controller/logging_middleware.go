package controller

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"time"
)

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
