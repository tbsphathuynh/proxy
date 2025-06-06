package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Logger wraps structured logging with OpenTelemetry integration
// Provides consistent logging interface across application components
// Automatically correlates logs with distributed traces for observability
// Time Complexity: O(1) for logging operations
// Space Complexity: O(1) per log entry
type Logger struct {
    slogger *slog.Logger // Structured logger implementation
    tracer  trace.Tracer // OpenTelemetry tracer for correlation
}

// LogLevel represents logging severity levels
// Maps to standard syslog levels for consistent interpretation
type LogLevel int

const (
    LogLevelDebug LogLevel = iota // Detailed debugging information
    LogLevelInfo                  // General information messages
    LogLevelWarn                  // Warning conditions
    LogLevelError                 // Error conditions
    LogLevelFatal                 // Critical errors causing termination
)

// NewLogger creates structured logger with OpenTelemetry integration
// Configures JSON output for structured log parsing and correlation
// Initializes tracer for distributed tracing integration
// Time Complexity: O(1) - logger initialisation
// Space Complexity: O(1) - fixed logger structure
func NewLogger(service string) *Logger {
    // Configure structured JSON logging
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
        AddSource: true,
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            // Rename timestamp field for consistency
            if a.Key == slog.TimeKey {
                a.Key = "timestamp"
            }
            return a
        },
    })

    logger := slog.New(handler)
    tracer := otel.Tracer(service)

    return &Logger{
        slogger: logger,
        tracer:  tracer,
    }
}

// Debug logs debug-level message with context and trace correlation
// Used for detailed debugging information in development/troubleshooting
// Automatically includes trace and span IDs when available
// Time Complexity: O(1) - structured logging with fixed overhead
// Space Complexity: O(n) where n is message and attribute size
func (l *Logger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
    l.logWithTrace(ctx, slog.LevelDebug, msg, attrs...)
}

// Info logs informational message with context and trace correlation
// Used for general application flow and business logic events
// Standard level for production operational logging
// Time Complexity: O(1) - structured logging with fixed overhead
// Space Complexity: O(n) where n is message and attribute size
func (l *Logger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
    l.logWithTrace(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs warning message with context and trace correlation
// Used for recoverable errors and unexpected conditions
// Indicates potential issues requiring attention
// Time Complexity: O(1) - structured logging with fixed overhead
// Space Complexity: O(n) where n is message and attribute size
func (l *Logger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
    l.logWithTrace(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs error message with context and trace correlation
// Used for application errors and exception conditions
// Automatically marks associated span as error for tracing
// Time Complexity: O(1) - structured logging with fixed overhead
// Space Complexity: O(n) where n is message and attribute size
func (l *Logger) Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
    // Add error to attributes if provided
    if err != nil {
        attrs = append(attrs, slog.String("error", err.Error()))
        
        // Mark span as error for distributed tracing
        if span := trace.SpanFromContext(ctx); span.IsRecording() {
            span.SetStatus(codes.Error, err.Error())
            span.RecordError(err)
        }
    }
    
    l.logWithTrace(ctx, slog.LevelError, msg, attrs...)
}

// Fatal logs fatal error and terminates application
// Used for unrecoverable errors requiring immediate shutdown
// Exits with code 1 after logging for monitoring systems
// Time Complexity: O(1) - logging followed by termination
// Space Complexity: O(n) where n is message and attribute size
func (l *Logger) Fatal(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
    if err != nil {
        attrs = append(attrs, slog.String("error", err.Error()))
    }
    
    l.logWithTrace(ctx, slog.LevelError, msg, attrs...)
    os.Exit(1)
}

// logWithTrace adds OpenTelemetry trace correlation to log entries
// Extracts trace and span IDs from context for log correlation
// Enables linking logs to distributed traces for debugging
// Time Complexity: O(1) - context extraction and logging
// Space Complexity: O(1) - adds fixed trace fields
func (l *Logger) logWithTrace(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
    // Extract trace information from context
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        attrs = append(attrs,
            slog.String("trace_id", span.SpanContext().TraceID().String()),
            slog.String("span_id", span.SpanContext().SpanID().String()),
        )
    }

    // Add service context information
    attrs = append(attrs,
        slog.String("service", "proxy"),
        slog.Time("timestamp", time.Now()),
    )

    l.slogger.LogAttrs(ctx, level, msg, attrs...)
}

// StartSpan creates new OpenTelemetry span with logging context
// Provides distributed tracing for request flow and performance monitoring
// Automatically propagates trace context for downstream services
// Time Complexity: O(1) - span creation and context propagation
// Space Complexity: O(1) - span metadata storage
func (l *Logger) StartSpan(ctx context.Context, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    return l.tracer.Start(ctx, operationName, trace.WithAttributes(attrs...))
}

// WithFields creates logger with pre-configured attributes
// Useful for adding consistent context to related log entries
// Returns new logger instance to avoid modifying original
    // Time Complexity: O(n) where n is number of attributes
    // Space Complexity: O(n) for attribute storage
func (l *Logger) WithFields(attrs ...slog.Attr) *Logger {
    anyAttrs := make([]any, len(attrs))
    for i, a := range attrs {
        anyAttrs[i] = a
    }
    return &Logger{
        slogger: l.slogger.With(anyAttrs...),
        tracer:  l.tracer,
    }
}

// HTTPRequestLogger creates middleware for HTTP request logging
// Logs request details including method, path, status, and duration
// Integrates with OpenTelemetry for distributed request tracing
// Time Complexity: O(1) per request
// Space Complexity: O(1) for request metadata
func (l *Logger) HTTPRequestLogger() func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Start span for request tracing
            ctx, span := l.StartSpan(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path),
                attribute.String("http.method", r.Method),
                attribute.String("http.url", r.URL.String()),
                attribute.String("http.user_agent", r.UserAgent()),
                attribute.String("http.remote_addr", r.RemoteAddr),
            )
            defer span.End()
            
            // Create response writer wrapper to capture status
            wrapper := &responseWriter{ResponseWriter: w, statusCode: 200}
            
            // Process request with tracing context
            next.ServeHTTP(wrapper, r.WithContext(ctx))
            
            duration := time.Since(start)
            
            // Log request completion with metrics
            l.Info(ctx, "HTTP request completed",
                slog.String("method", r.Method),
                slog.String("path", r.URL.Path),
                slog.Int("status", wrapper.statusCode),
                slog.Duration("duration", duration),
                slog.String("user_agent", r.UserAgent()),
                slog.String("remote_addr", r.RemoteAddr),
            )
            
            // Add span attributes for tracing
            span.SetAttributes(
                attribute.Int("http.status_code", wrapper.statusCode),
                attribute.String("http.response.duration", duration.String()),
            )
            
            // Mark span as error for 4xx/5xx responses
            if wrapper.statusCode >= 400 {
                span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapper.statusCode))
            }
        })
    }
}

// responseWriter wraps http.ResponseWriter to capture response status
// Used by HTTP logging middleware to record response codes
// Implements ResponseWriter interface transparently
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

// WriteHeader captures status code for logging
// Preserves original ResponseWriter behavior while recording status
func (w *responseWriter) WriteHeader(code int) {
    w.statusCode = code
    w.ResponseWriter.WriteHeader(code)
}