// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package main is a minimal example of inputstats usage.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gostafa/inputstats"
)

const (
	errFmtPrint   = "print: %w"
	errFmtStart   = "start: %w"
	errFmtStats   = "print stats: %w"
	errFmtWrap    = "%w"
	statsInterval = 5 * time.Second
	ten           = 10
	zero          = 0
)

func consume(statsCh <-chan inputstats.Stats, err error) error {
	if err != nil {
		return fmt.Errorf(errFmtStart, err)
	}

	outErr := printStats(statsCh)
	if outErr != nil {
		return fmt.Errorf(errFmtWrap, outErr)
	}

	return nil
}

func flushErr(out io.Writer, err error) {
	if err == nil {
		return
	}

	written, writeErr := io.WriteString(out, err.Error()+"\n")
	if writeErr != nil {
		return
	}

	if written == zero {
		return
	}
}

func formatStats(stats *inputstats.Stats) string {
	return "keys=" + strconv.FormatUint(stats.KeyboardClicks, ten) +
		" moves=" + strconv.FormatUint(stats.MouseMoves, ten) +
		" left=" + strconv.FormatUint(stats.LeftClicks, ten) +
		" right=" + strconv.FormatUint(stats.RightClicks, ten) +
		"\n"
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	report(run(ctx))
}

func printStats(statsCh <-chan inputstats.Stats) error {
	err := writeStats(os.Stdout, statsCh)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func report(err error) {
	flushErr(os.Stderr, err)
}

func run(ctx context.Context) error {
	err := consume(inputstats.Start(ctx, statsInterval))
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func writeLine(out io.Writer, line string) error {
	written, err := io.WriteString(out, line)
	if err != nil {
		return fmt.Errorf(errFmtStats, err)
	}

	if written == zero {
		return fmt.Errorf(errFmtStats, io.ErrShortWrite)
	}

	return nil
}

func writeStats(out io.Writer, statsCh <-chan inputstats.Stats) error {
	for stats := range statsCh {
		err := writeLine(out, formatStats(&stats))
		if err != nil {
			return fmt.Errorf(errFmtPrint, err)
		}
	}

	return nil
}
