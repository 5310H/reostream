package supervisor

import (
	"errors"
	"testing"
	"time"

	"github.com/VoltTech21/reostream/internal/config"
)

// LastError is kept after recovery, so it needs a timestamp.
func TestALastErrorCarriesWhenItHappened(t *testing.T) {
	s := New([]config.Camera{
		{Name: "fisheye", Address: "192.0.2.1", Streams: []string{"sub"}},
	}, nil)

	if got := s.Stats()[0]; got.LastError != "" || !got.LastErrorAt.IsZero() {
		t.Fatalf("a stream that has not failed reports error %q at %v",
			got.LastError, got.LastErrorAt)
	}

	before := time.Now()
	s.setResult(s.entries["fisheye/sub"], errors.New("held session"))
	after := time.Now()

	got := s.Stats()[0]
	if got.LastError != "held session" {
		t.Errorf("LastError = %q, want %q", got.LastError, "held session")
	}
	if got.LastErrorAt.Before(before) || got.LastErrorAt.After(after) {
		t.Errorf("LastErrorAt = %v, want a time between %v and %v",
			got.LastErrorAt, before, after)
	}

	// Recovery must not reset the error's time.
	was := got.LastErrorAt
	s.setRunning(s.entries["fisheye/sub"], true)
	if again := s.Stats()[0]; !again.LastErrorAt.Equal(was) {
		t.Errorf("LastErrorAt moved on recovery: %v, was %v", again.LastErrorAt, was)
	}
}
