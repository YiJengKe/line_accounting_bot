package logger

import (
	"accountingbot/config"
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

// Init initializes slog and OpenTelemetry
func Init() func(context.Context) error {
	var shutdownFunc func(context.Context) error

	cfg := config.Get()
	initOnce.Do(func() {
		tp, err := initTracer()
		if err != nil {
			slog.Error("Failed to initialize OpenTelemetry tracer", "error", err)
		} else {
			tracerProv = tp
			shutdownFunc = func(ctx context.Context) error {
				return tp.Shutdown(ctx)
			}
		}

		// Set slog handler
		var handler slog.Handler
		environment := cfg.Environment
		if environment == "production" {
			// Use JSON format in production
			opts := &slog.HandlerOptions{
				Level:       slog.LevelInfo,
				ReplaceAttr: addTraceInfo,
			}
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			// Use JSON format in development, but enable more detailed logs
			opts := &slog.HandlerOptions{
				Level:       slog.LevelDebug,
				ReplaceAttr: addTraceInfo,
			}
			handler = slog.NewJSONHandler(os.Stdout, opts)
		}

		logger = slog.New(handler)
		slog.SetDefault(logger)

		Info(context.Background(), "Logger and tracing system initialized")
	})

	return shutdownFunc
}

// initTracer initializes the OpenTelemetry tracer
func initTracer() (*sdktrace.TracerProvider, error) {
	cfg := config.Get()

	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(cfg.Trace.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// addTraceInfo adds trace_id and span_id to logs
func addTraceInfo(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return attr
	}

	if attr.Key == "trace_id" || attr.Key == "span_id" {
		return attr
	}

	return attr
}

// getSpanContext gets span info from context
func getSpanContext(ctx context.Context) trace.SpanContext {
	if ctx == nil {
		return trace.SpanContext{}
	}
	return trace.SpanFromContext(ctx).SpanContext()
}

// Debug logs DEBUG level messages
func Debug(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Debug(msg, args...)
}

// Info logs INFO level messages
func Info(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Info(msg, args...)
}

// Warn logs WARN level messages
func Warn(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Warn(msg, args...)
}

// Error logs ERROR level messages
func Error(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Error(msg, args...)
}

// Fatal logs ERROR level messages and exits the program
func Fatal(ctx context.Context, msg string, args ...any) {
	spanCtx := getSpanContext(ctx)
	if spanCtx.IsValid() {
		args = append(args, "trace_id", spanCtx.TraceID().String())
		args = append(args, "span_id", spanCtx.SpanID().String())
	}
	logger.Error(msg, args...)
	os.Exit(1)
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Get tracer
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	return tracer.Start(ctx, name)
}

// GetTracer gets a tracer instance
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(serviceName)
}
