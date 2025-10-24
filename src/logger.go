package godible

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		a.Value = slog.AnyValue(time.Now().Format(time.DateTime))
	}
	if a.Key == slog.SourceKey {
		source := a.Value.Any().(*slog.Source)
		fileName := filepath.Base(source.File)
		a_str := fmt.Sprintf("%s:%d", fileName, source.Line)
		a.Value = slog.StringValue(a_str)
	}
	return a
}

func SetDefaultLogger(level slog.Level) {
	jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}))
	slog.SetDefault(jsonLogger)
}
