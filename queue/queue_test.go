package queue

import (
	"testing"
)

func TestNew(t *testing.T) {
	bufferSize := 100
	q := New(bufferSize)

	if cap(q.DataChannel) != bufferSize {
		t.Errorf("New() DataChannel capacity = %d, want %d", cap(q.DataChannel), bufferSize)
	}
}

func TestQueue_Close(t *testing.T) {
	q := New(10)

	// Closing once should be fine
	q.Close()

	// Verify channels are closed
	_, ok := <-q.DataChannel
	if ok {
		t.Error("DataChannel should be closed, but it's not")
	}

	// Closing a second time should not panic
	q.Close()
}
