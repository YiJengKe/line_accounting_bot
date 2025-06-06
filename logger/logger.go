package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName = "line-accounting-bot"
)

var (
	logger     *slog.Logger
	initOnce   sync.Once
	tracerProv *sdktrace.TracerProvider
)

// Init 初始化 slog 和 OpenTelemetry
func Init() func(context.Context) error {
	var shutdownFunc func(context.Context) error

	initOnce.Do(func() {
		// 初始化追蹤器 (tracer)
		tp, err := initTracer()
		if err != nil {
			slog.Error("無法初始化 OpenTelemetry tracer", "error", err)
		} else {
			tracerProv = tp
			shutdownFunc = func(ctx context.Context) error {
				return tp.Shutdown(ctx)
			}
		}

		// 設定 slog 的 handler
		var handler slog.Handler
		environment := os.Getenv("ENVIRONMENT")
		if environment == "production" {
			// 生產環境使用 JSON 格式
			opts := &slog.HandlerOptions{
				Level: slog.LevelInfo,
				// 自訂 slog handler，加入 trace 與 span ID
				ReplaceAttr: addTraceInfo,
			}
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			// 開發環境也使用 JSON 格式，但啟用更詳細的紀錄
			opts := &slog.HandlerOptions{
				Level: slog.LevelDebug,
				// 自訂 slog handler，加入 trace 與 span ID
				ReplaceAttr: addTraceInfo,
			}
			handler = slog.NewJSONHandler(os.Stdout, opts)
		}

		// 建立及設定 logger
		logger = slog.New(handler)
		slog.SetDefault(logger)

		// 紀錄初始化完成
		Info(context.Background(), "日誌與追蹤系統已初始化")
	})

	return shutdownFunc
}

// initTracer 初始化 OpenTelemetry tracer
func initTracer() (*sdktrace.TracerProvider, error) {
	// 1. 取得 Jaeger collector endpoint
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317" // 預設本地 Jaeger endpoint
	}

	// 2. 建立 OTLP exporter
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// 3. 設定資源（服務名稱等基本資訊）
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("environment", os.Getenv("ENVIRONMENT")),
		),
	)
	if err != nil {
		return nil, err
	}

	// 4. 建立 trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// 5. 註冊為全域 trace provider
	otel.SetTracerProvider(tp)

	// 6. 設定傳播器（用於分散式追蹤）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// addTraceInfo 為日誌加入 trace_id 與 span_id
func addTraceInfo(groups []string, attr slog.Attr) slog.Attr {
	// 只處理根層級的屬性
	if len(groups) > 0 {
		return attr
	}

	// 如果是 trace_id 或 span_id，已經加入過了，不需重複處理
	if attr.Key == "trace_id" || attr.Key == "span_id" {
		return attr
	}

	return attr
}

// getSpanContext 從 context 獲取 span 資訊
func getSpanContext(ctx context.Context) trace.SpanContext {
	if ctx == nil {
		return trace.SpanContext{}
	}
	return trace.SpanFromContext(ctx).SpanContext()
}

// 日誌函數會自動加入 trace_id 與 span_id

// Debug 紀錄 DEBUG 級別日誌
func Debug(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Debug(msg, args...)
}

// Info 紀錄 INFO 級別日誌
func Info(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Info(msg, args...)
}

// Warn 紀錄 WARN 級別日誌
func Warn(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Warn(msg, args...)
}

// Error 紀錄 ERROR 級別日誌
func Error(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Error(msg, args...)
}

// Fatal 紀錄 ERROR 級別日誌並結束程式
func Fatal(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Error(msg, args...)
	os.Exit(1)
}

// StartSpan 開始一個新的 span
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	// 如果 ctx 為 nil，建立新的 context
	if ctx == nil {
		ctx = context.Background()
	}

	// 取得 tracer
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	return tracer.Start(ctx, name)
}

// GetTracer 取得 tracer 實例
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(serviceName)
}
