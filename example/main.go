package main

import (
	"context"
	"os"

	"github.com/franklad/logger"
)

func main() {
	log := logger.New(
		logger.WithLevel("trace"),
		logger.WithLogFormat("console"),
		logger.WithOutput(os.Stdout),
	).WithFields("app", "example", "version", "1.0.0")

	log.Trace("This is a trace message")
	log.Debug("This is a debug message")
	log.Info("This is an info message")
	log.Warn("This is a warning message")
	log.Error(nil, "This is an error message")

	log.SetLevel("info")
	log.Debug("This debug message will not be logged due to level change")
	log.Info("This is another info message after changing level")

	ctx := log.WithContext(context.Background())
	log = log.FromContext(ctx).WithFields("key", "value")
	log.Info("This message is from context with additional fields")

	log.SetLogFormat("json")
	log.Info("This message is now in JSON format")
}
