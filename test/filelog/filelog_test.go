package filelog

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/LinPr/go-bootstrap/otel"
	"github.com/LinPr/go-bootstrap/test/config"
	otelotel "go.opentelemetry.io/otel"
)

func TestOtelLoggerRotation(t *testing.T) {
	tmpDir := "./logs/"
	logFile := filepath.Join(tmpDir, "app.log")

	rotate := otel.Rotate{
		Filename:   logFile,
		MaxMB:      1, // 1 MB — small enough to trigger rotation quickly
		MaxBackups: 1,
		MaxDay:     1,
		LocalTime:  true,
		Compress:   true,
	}

	verifyRotation := func(t *testing.T) {
		t.Helper()

		info, err := os.Stat(logFile)
		if err != nil {
			t.Fatalf("log file not found: %v", err)
		}
		if info.Size() == 0 {
			t.Fatal("log file is empty")
		}
		t.Logf("main log file size: %d bytes", info.Size())

		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("failed to read temp dir: %v", err)
		}

		var backups []string
		for _, entry := range entries {
			if entry.Name() != filepath.Base(logFile) {
				backups = append(backups, entry.Name())
			}
		}
		t.Logf("backup files: %v", backups)
	}

	t.Run("sequential", func(t *testing.T) {
		config.SetupFilelogProvider(t, rotate)

		tracer := otelotel.Tracer("rotation-test-tracer")
		ctx, span := tracer.Start(context.Background(), "rotation-test-root")
		defer span.End()

		for i := 0; i < 1; i++ {
			slog.InfoContext(ctx, "rotation test log message", "iteration", i)
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := otel.ShutdownOtelProvider(shutdownCtx); err != nil {
			t.Fatalf("failed to shutdown provider: %v", err)
		}

		verifyRotation(t)
	})

	t.Run("concurrent", func(t *testing.T) {
		config.SetupFilelogProvider(t, rotate)

		tracer := otelotel.Tracer("rotation-concurrent-tracer")
		ctx, span := tracer.Start(context.Background(), "rotation-concurrent-root")
		defer span.End()

		const goroutines = 8
		const logsPerGoroutine = 1000

		var wg sync.WaitGroup
		for g := range goroutines {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < logsPerGoroutine; i++ {
					slog.InfoContext(ctx, "concurrent rotation log message",
						"goroutine", goroutineID, "iteration", i)
				}
			}(g)
		}
		wg.Wait()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := otel.ShutdownOtelProvider(shutdownCtx); err != nil {
			t.Fatalf("failed to shutdown provider: %v", err)
		}

		verifyRotation(t)
	})
}

func TestWriteLog(t *testing.T) {
	tmpDir := "./logs/"
	logFile := filepath.Join(tmpDir, "app.log")
	data := []byte(`{"resourceLogs":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"go-bootstrap"}},{"key":"service.version","value":{"stringValue":"1.0.0"}}]},"scopeLogs":[{"scope":{"name":"global"},"logRecords":[{"timeUnixNano":"1789538527305824828","observedTimeUnixNano":"1789538527305835929","severityNumber":"SEVERITY_NUMBER_INFO","severityText":"INFO","body":{"stringValue":"log provider initialized"},"attributes":[{"key":"code.file.path","value":{"stringValue":"/home/lin/C++42/Go/go-bootstrap/otel/provider.go"}},{"key":"code.function.name","value":{"stringValue":"github.com/LinPr/go-bootstrap/otel.(*OtelProviders).initLog"}},{"key":"code.line.number","value":{"intValue":"198"}},{"key":"exporter","value":{"stringValue":"file"}},{"key":"logger","value":{"stringValue":"slog"}}]},{"timeUnixNano":"1789538527305907244","observedTimeUnixNano":"1789538527305912254","severityNumber":"SEVERITY_NUMBER_INFO","severityText":"INFO","body":{"stringValue":"rotation test log message"},"attributes":[{"key":"code.file.path","value":{"stringValue":"/home/lin/C++42/Go/go-bootstrap/test/filelog/filelog_test.go"}},{"key":"code.function.name","value":{"stringValue":"github.com/LinPr/go-bootstrap/test/filelog.TestOtelLoggerRotation.func2"}},{"key":"code.line.number","value":{"intValue":"64"}},{"key":"iteration","value":{"intValue":"0"}}]}]}]}]}`)
	data = append(data, '\n')
	if err := os.WriteFile(logFile, data, 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}
}
