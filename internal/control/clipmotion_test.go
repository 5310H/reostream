package control

import (
	"sync"
	"testing"
	"time"

	"github.com/VoltTech21/reostream/internal/alarms"
)

// A battery camera has no live alert connection; the files it uploads are
// its motion. One ON for a burst of files, one OFF after the last.
func TestUploadedClipsAreMotionForACameraWithNoAlertConnection(t *testing.T) {
	hold := clipMotionHold
	clipMotionHold = 80 * time.Millisecond
	t.Cleanup(func() { clipMotionHold = hold })

	var mu sync.Mutex
	var got []alarms.Change
	r := &eventRelay{
		handle:  func(c alarms.Change) { mu.Lock(); got = append(got, c); mu.Unlock() },
		watched: map[string]bool{"driveway": true},
	}
	r.clipArrived("porch")
	time.Sleep(30 * time.Millisecond)
	r.clipArrived("porch") // the clip after the snapshot
	r.clipArrived("driveway")
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || !got[0].On || got[1].On || got[0].Camera != "porch" || got[0].Kind != "motion" {
		t.Fatalf("got %+v, want porch motion ON then OFF only", got)
	}
}

func TestNoClipMotionWhileTheRelayIsOff(t *testing.T) {
	r := &eventRelay{}
	r.clipArrived("porch") // must not panic or start a timer
	if len(r.clips) != 0 {
		t.Fatal("a timer started with the relay off")
	}
}
