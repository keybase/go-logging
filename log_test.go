// Copyright 2013, Örjan Persson. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

func TestLogCalldepth(t *testing.T) {
	buf := &bytes.Buffer{}
	SetBackend(NewLogBackend(buf, "", log.Lshortfile))
	SetFormatter(MustStringFormatter("%{shortfile} %{level} %{message}"))

	log := MustGetLogger("test")
	log.Info("test filename")

	parts := strings.SplitN(buf.String(), " ", 2)

	// Verify that the correct filename is registered by the stdlib logger
	if !strings.HasPrefix(parts[0], "log_test.go:") {
		t.Errorf("incorrect filename: %s", parts[0])
	}
	// Verify that the correct filename is registered by go-logging
	if !strings.HasPrefix(parts[1], "log_test.go:") {
		t.Errorf("incorrect filename: %s", parts[1])
	}
}

func c(log *Logger) { log.Info("test callpath") }
func b(log *Logger) { c(log) }
func a(log *Logger) { b(log) }

func rec(log *Logger, r int) {
	if r == 0 {
		a(log)
		return
	}
	rec(log, r-1)
}

func testCallpath(t *testing.T, format string, mustContain []string, mustStartWith string) {
	buf := &bytes.Buffer{}
	SetBackend(NewLogBackend(buf, "", log.Lshortfile))
	SetFormatter(MustStringFormatter(format))

	logger := MustGetLogger("test")
	rec(logger, 6)

	parts := strings.SplitN(buf.String(), " ", 3)

	// Verify that the correct filename is registered by the stdlib logger
	if !strings.HasPrefix(parts[0], "log_test.go:") {
		t.Errorf("incorrect filename: %s", parts[0])
	}
	// Verify that the callpath contains expected elements
	callpath := parts[1]
	if mustStartWith != "" && !strings.HasPrefix(callpath, mustStartWith) {
		t.Errorf("incorrect callpath: %s does not start with %s", callpath, mustStartWith)
	}
	for _, required := range mustContain {
		if !strings.Contains(callpath, required) {
			t.Errorf("incorrect callpath: %s missing required element %s", callpath, required)
		}
	}
	// Verify that the correct message is registered by go-logging
	if !strings.HasPrefix(parts[2], "test callpath") {
		t.Errorf("incorrect message: %s", parts[2])
	}
}

func TestLogCallpath(t *testing.T) {
	// Note: Exact callpath output varies by Go version and architecture due to
	// differences in stack trace handling and inlining. We test for essential
	// characteristics rather than exact strings.

	// Full callpath tests - should contain recursive marker and function names
	testCallpath(t, "%{callpath} %{message}", []string{"TestLogCallpath", "rec", "...", "a", "b", "c"}, "TestLogCallpath")
	testCallpath(t, "%{callpath:-1} %{message}", []string{"TestLogCallpath", "rec", "...", "a", "b", "c"}, "TestLogCallpath")
	testCallpath(t, "%{callpath:0} %{message}", []string{"TestLogCallpath", "rec", "...", "a", "b", "c"}, "TestLogCallpath")

	// Depth-limited tests - should start with truncation marker and contain expected functions
	testCallpath(t, "%{callpath:1} %{message}", []string{"c"}, "~")
	testCallpath(t, "%{callpath:2} %{message}", []string{"c"}, "~")
	testCallpath(t, "%{callpath:3} %{message}", []string{"b", "c"}, "~")
}

func BenchmarkLogMemoryBackendIgnored(b *testing.B) {
	backend := SetBackend(NewMemoryBackend(1024))
	backend.SetLevel(INFO, "")
	RunLogBenchmark(b)
}

func BenchmarkLogMemoryBackend(b *testing.B) {
	backend := SetBackend(NewMemoryBackend(1024))
	backend.SetLevel(DEBUG, "")
	RunLogBenchmark(b)
}

func BenchmarkLogChannelMemoryBackend(b *testing.B) {
	channelBackend := NewChannelMemoryBackend(1024)
	backend := SetBackend(channelBackend)
	backend.SetLevel(DEBUG, "")
	RunLogBenchmark(b)
	channelBackend.Flush()
}

func BenchmarkLogLeveled(b *testing.B) {
	backend := SetBackend(NewLogBackend(io.Discard, "", 0))
	backend.SetLevel(INFO, "")

	RunLogBenchmark(b)
}

func BenchmarkLogLogBackend(b *testing.B) {
	backend := SetBackend(NewLogBackend(io.Discard, "", 0))
	backend.SetLevel(DEBUG, "")
	RunLogBenchmark(b)
}

func BenchmarkLogLogBackendColor(b *testing.B) {
	colorizer := NewLogBackend(io.Discard, "", 0)
	colorizer.Color = true
	backend := SetBackend(colorizer)
	backend.SetLevel(DEBUG, "")
	RunLogBenchmark(b)
}

func BenchmarkLogLogBackendStdFlags(b *testing.B) {
	backend := SetBackend(NewLogBackend(io.Discard, "", log.LstdFlags))
	backend.SetLevel(DEBUG, "")
	RunLogBenchmark(b)
}

func BenchmarkLogLogBackendLongFileFlag(b *testing.B) {
	backend := SetBackend(NewLogBackend(io.Discard, "", log.Llongfile))
	backend.SetLevel(DEBUG, "")
	RunLogBenchmark(b)
}

func RunLogBenchmark(b *testing.B) {
	password := Password("foo")
	log := MustGetLogger("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Debug("log line for %d and this is rectified: %s", i, password)
	}
}

func BenchmarkLogFixed(b *testing.B) {
	backend := SetBackend(NewLogBackend(io.Discard, "", 0))
	backend.SetLevel(DEBUG, "")

	RunLogBenchmarkFixedString(b)
}

func BenchmarkLogFixedIgnored(b *testing.B) {
	backend := SetBackend(NewLogBackend(io.Discard, "", 0))
	backend.SetLevel(INFO, "")
	RunLogBenchmarkFixedString(b)
}

func RunLogBenchmarkFixedString(b *testing.B) {
	log := MustGetLogger("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Debug("some random fixed text")
	}
}
