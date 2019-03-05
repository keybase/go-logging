// +build !windows

// Copyright 2013, Örjan Persson. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/eapache/channels"
)

const (
	// TODO: comment and tune these parameters
	logTimesKept = 4

	logGap = 10 * time.Millisecond
)

// LogBackend utilizes the standard log module.
type LogBackend struct {
	Logger       *log.Logger
	Color        bool
	ColorConfig  []string
	lastLogTimes [logTimesKept]time.Time
	logChannel   *channels.InfiniteChannel
}

func (b *LogBackend) loop() {
	for logMsgUntyped := range b.logChannel.Out() {
		logMsg := logMsgUntyped.(string)
		now := time.Now()

		// If we have logged at least n times in the last m milliseconds,
		//  then throttle the logging and condense it into fewer syscalls.
		if now.Sub(b.lastLogTimes[logTimesKept-1]) < logGap {
			buf := &bytes.Buffer{}
			buf.WriteString(logMsg)

		Inner:
			for {
				var nextMessageUntyped interface{}
				select {
				case nextMessageUntyped = <-b.logChannel.Out():
					nextMessage := nextMessageUntyped.(string)
					buf.WriteString("\n")
					buf.WriteString(nextMessage)
				case <-time.After(logGap):
					// TODO: preserve the call depth
					err := b.Logger.Output(2, buf.String())
					if err != nil {
						// TODO: something better
						panic(err)
					}
					break Inner
				}
			}
		} else { // our logging rate is low - we can just log the message

			// TODO: preserve the call depth
			err := b.Logger.Output(2, logMsg)
			if err != nil {
				// TODO: something better
				panic(err)
			}
		}
		for i := logTimesKept - 1; i > 0; i-- {
			b.lastLogTimes[i] = b.lastLogTimes[i-1]
		}
		// because now was recorded at the beginning of the function,
		// if we took the delay case then we are allowing n more logs before
		// we delay again.
		b.lastLogTimes[0] = now
	}
}

// NewLogBackend creates a new LogBackend.
func NewLogBackend(out io.Writer, prefix string, flag int) *LogBackend {
	b := &LogBackend{Logger: log.New(out, prefix, flag)}
	// TODO: consider instead having a fixed channel size,
	//  and if that channel gets full then trigger a clear of the buffer
	b.logChannel = channels.NewInfiniteChannel()
	go b.loop()
	return b
	// TODO: have a shutdown on this
}

// Log implements the Backend interface.
func (b *LogBackend) Log(level Level, calldepth int, rec *Record) error {
	if b.Color {
		col := colors[level]
		if len(b.ColorConfig) > int(level) && b.ColorConfig[level] != "" {
			col = b.ColorConfig[level]
		}

		buf := &bytes.Buffer{}
		buf.Write([]byte(col))
		buf.Write([]byte(rec.Formatted(calldepth + 1)))
		buf.Write([]byte("\033[0m"))
		// For some reason, the Go logger arbitrarily decided "2" was the correct
		// call depth...
		// TODO: deal with errors
		b.logChannel.In() <- buf.String()
		return nil
	}

	b.logChannel.In() <- rec.Formatted(calldepth + 1)
	return nil
}
