package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gostafa/inputstats"
)

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) {
	return 0, nil
}

func TestWriteLineEmpty(t *testing.T) {
	err := writeLine(zeroWriter{}, "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFlushErrEmpty(t *testing.T) {
	flushErr(zeroWriter{}, errors.New("boom"))
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestConsumeError(t *testing.T) {
	err := consume(nil, errors.New("denied"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConsumePrints(t *testing.T) {
	ch := make(chan inputstats.Stats)
	close(ch)

	err := consume(ch, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
}

func TestFlushErrNil(t *testing.T) {
	flushErr(io.Discard, nil)
}

func TestFlushErrWrite(t *testing.T) {
	flushErr(io.Discard, errors.New("boom"))
}

func TestFlushErrWriteFail(t *testing.T) {
	flushErr(failWriter{}, errors.New("boom"))
}

func TestFormatStats(t *testing.T) {
	got := formatStats(
		&inputstats.Stats{KeyboardClicks: 1, MouseMoves: 2, LeftClicks: 3, RightClicks: 4},
	)
	want := "keys=1 moves=2 left=3 right=4\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMainInterrupt(t *testing.T) {
	go func() {
		time.Sleep(50 * time.Millisecond)
		err := syscall.Kill(os.Getpid(), syscall.SIGINT)
		if err != nil {
			t.Errorf("kill: %v", err)
		}
	}()

	main()
}

func TestConsumePrintError(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	orig := os.Stdout
	os.Stdout = writer

	defer func() {
		os.Stdout = orig
		closeErr := reader.Close()
		if closeErr != nil {
			t.Errorf("reader: %v", closeErr)
		}
	}()

	ch := make(chan inputstats.Stats, 1)
	ch <- inputstats.Stats{}
	close(ch)

	if consume(ch, nil) == nil {
		t.Fatal("expected error")
	}
}

func TestPrintStatsError(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	orig := os.Stdout
	os.Stdout = writer

	defer func() {
		os.Stdout = orig
		closeErr := reader.Close()
		if closeErr != nil {
			t.Errorf("reader: %v", closeErr)
		}
	}()

	ch := make(chan inputstats.Stats, 1)
	ch <- inputstats.Stats{}
	close(ch)

	if printStats(ch) == nil {
		t.Fatal("expected error")
	}
}

func TestPrintStats(t *testing.T) {
	ch := make(chan inputstats.Stats)
	close(ch)

	err := printStats(ch)
	if err != nil {
		t.Fatalf("printStats: %v", err)
	}
}

func TestReport(t *testing.T) {
	report(nil)
	report(errors.New("shown"))
}

func TestRunCancel(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "allow")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Log(err)
	}
}

func TestRunError(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "deny")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunOK(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "allow")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("run: %v", err)
	}
}

func TestWriteLineError(t *testing.T) {
	err := writeLine(failWriter{}, "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteLineOK(t *testing.T) {
	var buf bytes.Buffer

	err := writeLine(&buf, "x")
	if err != nil {
		t.Fatalf("writeLine: %v", err)
	}
}

func TestWriteStatsError(t *testing.T) {
	ch := make(chan inputstats.Stats, 1)
	ch <- inputstats.Stats{}
	close(ch)

	err := writeStats(failWriter{}, ch)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteStatsOK(t *testing.T) {
	ch := make(chan inputstats.Stats, 1)
	ch <- inputstats.Stats{KeyboardClicks: 1}
	close(ch)

	var buf bytes.Buffer

	err := writeStats(&buf, ch)
	if err != nil {
		t.Fatalf("writeStats: %v", err)
	}
}
