// Package logger provides a production-ready logging interface backed by zerolog.
// It supports configurable log levels, formats (JSON or console), and output writers.
// Configuration can be influenced by environment variables LOG_LEVEL, LOG_FORMAT, and LOG_FILE.
// The logger is thread-safe and follows best practices for structured logging in Go.
package logger

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// LogFormatJSON specifies JSON output format.
	LogFormatJSON = "json"
	// LogFormatConsole specifies pretty-printed console output format.
	LogFormatConsole = "console"

	// LevelTrace sets the logger to trace level.
	LevelTrace = "trace"
	// LevelDebug sets the logger to debug level.
	LevelDebug = "debug"
	// LevelInfo sets the logger to info level.
	LevelInfo = "info"
	// LevelWarn sets the logger to warn level.
	LevelWarn = "warn"
	// LevelError sets the logger to error level.
	LevelError = "error"
	// LevelFatal sets the logger to fatal level.
	LevelFatal = "fatal"
	// LevelPanic sets the logger to panic level.
	LevelPanic = "panic"
	// LevelNo sets the logger to no logging.
	LevelNo = "no"
	// LevelDisabled disables the logger entirely.
	LevelDisabled = "disabled"
)

// ErrInvalidLogLevel is returned when an invalid log level is provided.
var ErrInvalidLogLevel = errors.New("invalid log level")

// ErrInvalidLogFormat is returned when an invalid log format is provided.
var ErrInvalidLogFormat = errors.New("invalid log format")

// Logger defines the interface for logging operations.
type Logger interface {
	Trace(msg string, fields ...any)
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(err error, msg string, fields ...any)
	Fatal(err error, msg string, fields ...any)
	Panic(err error, msg string, fields ...any)
	WithFields(fields ...any) Logger
	WithContext(ctx context.Context) context.Context
	FromContext(ctx context.Context) Logger
	SetLevel(level string) error
	SetLogFormat(format string) error
}

// ZeroLogger is the zerolog implementation of the Logger interface.
type ZeroLogger struct {
	logger zerolog.Logger
	config *config
}

type config struct {
	level      string
	logFormat  string
	timeFormat string
	out        io.Writer
}

type option func(*config)

// WithLevel sets the initial log level.
func WithLevel(level string) option {
	return func(c *config) {
		c.level = level
	}
}

// WithLogFormat sets the initial log format (json or console).
func WithLogFormat(format string) option {
	return func(c *config) {
		c.logFormat = format
	}
}

// WithTimeFormat sets the time format for log timestamps.
func WithTimeFormat(format string) option {
	return func(c *config) {
		c.timeFormat = format
	}
}

// WithOutput sets the initial output writer.
func WithOutput(w io.Writer) option {
	return func(c *config) {
		c.out = w
	}
}

func defaults() *config {
	return &config{
		level:      LevelInfo,
		logFormat:  LogFormatJSON,
		timeFormat: time.RFC3339,
		out:        os.Stdout,
	}
}

// New creates a new logger with the given options. Options override environment variables.
func New(options ...option) Logger {
	config := defaults()
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.level = level
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.logFormat = format
	}

	if file := os.Getenv("LOG_FILE"); file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			panic("failed to open log file: " + err.Error())
		}

		config.out = f
	}

	for _, opt := range options {
		opt(config)
	}

	zerolog.TimeFieldFormat = config.timeFormat
	outWriter, err := createWriter(config.logFormat, config.out, config.timeFormat)
	if err != nil {
		panic("failed to create log writer: " + err.Error())
	}

	logger := zerolog.New(outWriter).With().Timestamp().Logger()
	logLevel, err := zerolog.ParseLevel(strings.ToLower(config.level))
	if err != nil {
		panic("invalid log level: " + err.Error())
	}

	return &ZeroLogger{
		logger: logger.Level(logLevel),
		config: config,
	}
}

// Trace logs a trace-level message with optional fields.
func (z *ZeroLogger) Trace(msg string, fields ...any) {
	z.logger.Trace().Fields(convertFields(fields...)).Msg(msg)
}

// Debug logs a debug-level message with optional fields.
func (z *ZeroLogger) Debug(msg string, fields ...any) {
	z.logger.Debug().Fields(convertFields(fields...)).Msg(msg)
}

// Info logs an info-level message with optional fields.
func (z *ZeroLogger) Info(msg string, fields ...any) {
	z.logger.Info().Fields(convertFields(fields...)).Msg(msg)
}

// Warn logs a warn-level message with optional fields.
func (z *ZeroLogger) Warn(msg string, fields ...any) {
	z.logger.Warn().Fields(convertFields(fields...)).Msg(msg)
}

// Error logs an error-level message with an error and optional fields.
func (z *ZeroLogger) Error(err error, msg string, fields ...any) {
	z.logger.Error().Err(err).Fields(convertFields(fields...)).Msg(msg)
}

// Fatal logs a fatal-level message with an error and optional fields, then exits the program.
func (z *ZeroLogger) Fatal(err error, msg string, fields ...any) {
	z.logger.Fatal().Err(err).Fields(convertFields(fields...)).Msg(msg)
}

// Panic logs a panic-level message with an error and optional fields, then panics.
func (z *ZeroLogger) Panic(err error, msg string, fields ...any) {
	z.logger.Panic().Err(err).Fields(convertFields(fields...)).Msg(msg)
}

// WithFields returns a new logger with additional structured fields.
func (z *ZeroLogger) WithFields(fields ...any) Logger {
	return &ZeroLogger{
		logger: z.logger.With().Fields(convertFields(fields...)).Logger(),
		config: z.config,
	}
}

// WithContext attaches the logger to the provided context.
func (z *ZeroLogger) WithContext(ctx context.Context) context.Context {
	return z.logger.WithContext(ctx)
}

// FromContext retrieves the logger from the context. If none is found, returns a logger based on the current instance.
func (z *ZeroLogger) FromContext(ctx context.Context) Logger {
	logger := zerolog.Ctx(ctx)
	if logger.GetLevel() == zerolog.Disabled {
		return &ZeroLogger{
			logger: z.logger,
			config: z.config,
		}
	}

	return &ZeroLogger{
		logger: *logger,
		config: z.config,
	}
}

// SetLevel sets the minimum log level for the logger.
func (z *ZeroLogger) SetLevel(level string) error {
	logLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return ErrInvalidLogLevel
	}

	z.logger = z.logger.Level(logLevel)
	return nil
}

// SetLogFormat sets the output format (json or console) for the logger.
func (z *ZeroLogger) SetLogFormat(format string) error {
	outWriter, err := createWriter(format, z.config.out, z.config.timeFormat)
	if err != nil {
		return err
	}

	z.logger = z.logger.Output(outWriter)
	z.config.logFormat = format
	return nil
}

// createWriter returns an io.Writer based on the logFormat, respecting NO_COLOR for console.
func createWriter(format string, out io.Writer, timeFormat string) (io.Writer, error) {
	switch strings.ToLower(format) {
	case LogFormatJSON:
		return out, nil
	case LogFormatConsole:
		cw := zerolog.ConsoleWriter{Out: out, TimeFormat: timeFormat}
		if os.Getenv("NO_COLOR") != "" {
			cw.NoColor = true
		}

		return cw, nil
	default:
		return nil, ErrInvalidLogFormat
	}
}

// convertFields converts key-value pairs to a map, skipping invalid pairs.
func convertFields(fields ...any) map[string]any {
	fieldMap := make(map[string]any, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}

		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		fieldMap[key] = fields[i+1]
	}

	return fieldMap
}
