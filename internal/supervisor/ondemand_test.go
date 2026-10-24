package supervisor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VoltTech21/reostream/internal/config"
	"github.com/VoltTech21/reostream/internal/hub"
	"github.com/VoltTech21/reostream/internal/stream"
)

// fastWake shortens the battery limits for a test and puts them back.
func fastWake(t *testing.T) {
	t.Helper()
	idle, max, cool, poll, retry := wakeIdle, wakeMax, wakeCooldown, wakePoll, wakeRetry
	sw := stillWait
	stillWait = map[string]time.Duration{"sub": 0, "main": 0}
	t.Cleanup(func() { stillWait = sw })
	wakeIdle, wakeMax, wakeCooldown, wakePoll, wakeRetry = 100*time.Millisecond, 400*time.Millisecond, 50*time.Millisecond, 10*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { wakeIdle, wakeMax, wakeCooldown, wakePoll, wakeRetry = idle, max, cool, poll, retry })
}

var battery = config.Camera{Name: "porch", Address: "192.0.2.10", UID: "95270000TESTUID1"}

// counting runs a session until it is cancelled, counting wakes and
// tracking whether one is in progress.
func counting(wakes, awake *int32) Runner {
	return func(ctx context.Context, cfg stream.Config, h *hub.Hub) error {
		atomic.AddInt32(wakes, 1)
		atomic.StoreInt32(awake, 1)
		<-ctx.Done()
		atomic.StoreInt32(awake, 0)
		return nil
	}
}

func TestBatteryCameraSleepsWithNoViewer(t *testing.T) {
	fastWake(t)
	var wakes, awake int32
	s := New([]config.Camera{battery}, counting(&wakes, &awake))
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	s.Run(ctx)
	if wakes != 0 {
		t.Fatalf("woke %d times with nobody watching", wakes)
	}
	h, ok := s.Hub("porch/sub")
	if !ok || !h.OnDemand() {
		t.Fatal("battery camera has no on-demand sub hub")
	}
}

func TestBatteryCameraWakesForAViewerAndSleepsAfter(t *testing.T) {
	fastWake(t)
	var wakes, awake int32
	s := New([]config.Camera{battery}, counting(&wakes, &awake))
	start(t, s)

	h, _ := s.Hub("porch/sub")
	_, unsub := h.Subscribe()
	waitFor(t, func() bool { return atomic.LoadInt32(&awake) == 1 })
	unsub()
	waitFor(t, func() bool { return atomic.LoadInt32(&awake) == 0 })
	if n := atomic.LoadInt32(&wakes); n != 1 {
		t.Fatalf("woke %d times for one viewer", n)
	}
}

// A viewer that never leaves still lets the camera sleep at wakeMax, and
// is disconnected so it does not wake it straight back up.
func TestBatteryCameraSleepsAtTheSessionLimit(t *testing.T) {
	fastWake(t)
	var wakes, awake int32
	s := New([]config.Camera{battery}, counting(&wakes, &awake))
	start(t, s)

	h, _ := s.Hub("porch/sub")
	ch, unsub := h.Subscribe()
	defer unsub()
	waitFor(t, func() bool { return atomic.LoadInt32(&awake) == 1 })
	select {
	case _, open := <-ch:
		for open {
			_, open = <-ch
		}
	case <-time.After(2 * time.Second):
		t.Fatal("viewer was never disconnected")
	}
	if atomic.LoadInt32(&awake) != 0 {
		t.Fatal("camera still awake after the session limit")
	}
	time.Sleep(200 * time.Millisecond)
	if n := atomic.LoadInt32(&wakes); n != 1 {
		t.Fatalf("woke %d times; the session limit should end it", n)
	}
}

// A camera that will not wake is tried a few times, then the viewer is
// let go rather than retried forever.
func TestBatteryCameraGivesUpOnAFailedWake(t *testing.T) {
	fastWake(t)
	var tries int32
	s := New([]config.Camera{battery}, func(ctx context.Context, cfg stream.Config, h *hub.Hub) error {
		atomic.AddInt32(&tries, 1)
		return errors.New("no answer")
	})
	start(t, s)

	h, _ := s.Hub("porch/sub")
	ch, unsub := h.Subscribe()
	defer unsub()
	select {
	case _, open := <-ch:
		if open {
			t.Fatal("got data from a camera that never woke")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("viewer kept waiting on a camera that will not wake")
	}
	if n := atomic.LoadInt32(&tries); n != int32(wakeTries) {
		t.Fatalf("tried %d times, want %d", n, wakeTries)
	}
}

func TestBatteryStreamsGetNoRTSP(t *testing.T) {
	cam := battery
	cam.RTSP = []string{"sub", "main"}
	s := New([]config.Camera{cam}, counting(new(int32), new(int32)))
	s.AttachSinks(func(camera, st string) stream.FrameSink {
		t.Fatalf("RTSP sink built for battery stream %s/%s", camera, st)
		return nil
	})
}

// start runs s until the test ends, and waits for it to stop before the
// test's cleanups, fastWake's included, run.
func start(t *testing.T, s *Supervisor) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
}

// A battery camera's last picture outlives a restart: kept after a wake,
// put back when the next process starts.
func TestBatteryStillSurvivesARestart(t *testing.T) {
	fastWake(t)
	dir := t.TempDir()
	key := []byte{0, 0, 0, 1, 0x65, 0x88}
	run := func(ctx context.Context, cfg stream.Config, h *hub.Hub) error {
		h.SetKeyframe("H264", key)
		<-ctx.Done()
		return nil
	}
	s := New([]config.Camera{battery}, run)
	s.SetStillDir(dir)
	start(t, s)
	h, _ := s.Hub("porch/sub")
	_, unsub := h.Subscribe()
	waitFor(t, func() bool { _, es := h.Keyframe(); return es != nil })
	unsub()
	waitFor(t, func() bool { _, err := os.Stat(filepath.Join(dir, "porch_sub.still")); return err == nil })

	again := New([]config.Camera{battery}, run)
	again.SetStillDir(dir)
	h2, _ := again.Hub("porch/sub")
	codec, es := h2.Keyframe()
	if codec != "H264" || string(es) != string(key) || h2.KeyframeAt().IsZero() {
		t.Fatalf("restored %q %v", codec, es)
	}
}

// With nothing to show, a battery camera is woken once, just until its
// first picture, which is kept; with a picture already, never.
func TestFirstPictureIsTakenOnceAndKept(t *testing.T) {
	fastWake(t)
	dir := t.TempDir()
	var wakes int32
	run := func(ctx context.Context, cfg stream.Config, h *hub.Hub) error {
		atomic.AddInt32(&wakes, 1)
		h.SetKeyframe("H264", []byte{0, 0, 0, 1, 0x65})
		<-ctx.Done()
		return nil
	}
	s := New([]config.Camera{battery}, run)
	s.SetStillDir(dir)
	start(t, s)
	waitFor(t, func() bool {
		_, e1 := os.Stat(filepath.Join(dir, "porch_sub.still"))
		_, e2 := os.Stat(filepath.Join(dir, "porch_main.still"))
		return e1 == nil && e2 == nil
	})
	time.Sleep(100 * time.Millisecond)
	if n := atomic.LoadInt32(&wakes); n != 2 {
		t.Fatalf("woke %d times, want once per stream", n)
	}

	var again int32
	s2 := New([]config.Camera{battery}, func(ctx context.Context, cfg stream.Config, h *hub.Hub) error {
		atomic.AddInt32(&again, 1)
		<-ctx.Done()
		return nil
	})
	s2.SetStillDir(dir)
	start(t, s2)
	time.Sleep(200 * time.Millisecond)
	if n := atomic.LoadInt32(&again); n != 0 {
		t.Fatalf("woke %d times with pictures already kept", n)
	}
}
