package util

import (
	"bytes"
	"testing"
	"time"
)

func TestProgress_Increment(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(100, &buf, "Testing...")

	// Verify initial state
	if p.Current() != 0 {
		t.Errorf("expected initial count 0, got %d", p.Current())
	}

	// Increment and verify
	p.Increment()
	if p.Current() != 1 {
		t.Errorf("expected count 1 after increment, got %d", p.Current())
	}

	// Multiple increments
	for i := 0; i < 49; i++ {
		p.Increment()
	}
	if p.Current() != 50 {
		t.Errorf("expected count 50, got %d", p.Current())
	}
}

func TestProgress_ConcurrentIncrement(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(1000, &buf, "Testing...")

	// Spawn multiple goroutines to increment concurrently
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				p.Increment()
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if p.Current() != 1000 {
		t.Errorf("expected count 1000 after concurrent increments, got %d", p.Current())
	}
}

func TestProgress_StartStop(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(10, &buf, "Testing...")

	p.Start()

	// Increment a few times
	for i := 0; i < 5; i++ {
		p.Increment()
		time.Sleep(10 * time.Millisecond)
	}

	p.Stop()

	// Verify something was written to the buffer
	if buf.Len() == 0 {
		t.Error("expected progress output, got empty buffer")
	}
}

func TestIsTTY_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	if IsTTY(&buf) {
		t.Error("expected bytes.Buffer to not be a TTY")
	}
}
