package rtsp

import (
	"sync"
	"testing"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/pion/rtp"
)

// The camera clock wraps every 71.58 minutes; RTP timestamps must not.
func TestRTPTimestampsSurviveTheCameraClockWrap(t *testing.T) {
	frames := captureFrames(t)
	srv, st := serverWithStream(t, "wrap")
	feedUntilReady(t, st, frames)

	tr := gortsplib.ProtocolTCP
	c := &gortsplib.Client{Protocol: &tr}
	u, err := base.ParseURL("rtsp://" + srv.Addr() + "/wrap")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	c.Scheme, c.Host = u.Scheme, u.Host
	if err := c.Start(); err != nil {
		t.Fatalf("client start: %v", err)
	}
	defer c.Close()

	desc, _, err := c.Describe(u)
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if err := c.SetupAll(desc.BaseURL, desc.Medias); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Count timestamps, not packets (a keyframe spans many packets).
	var mu sync.Mutex
	var stamps []uint32
	enough := make(chan struct{})
	var closed bool
	c.OnPacketRTPAny(func(_ *description.Media, f format.Format, p *rtp.Packet) {
		// Video only; audio has its own clock.
		switch f.(type) {
		case *format.H265, *format.H264:
		default:
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if len(stamps) > 0 && stamps[len(stamps)-1] == p.Timestamp {
			return
		}
		stamps = append(stamps, p.Timestamp)
		if len(stamps) >= 10 && !closed {
			closed = true
			close(enough)
		}
	})
	if _, err := c.Play(nil); err != nil {
		t.Fatalf("play: %v", err)
	}

	// Step the camera clock across the wrap using real frames.
	const step = 33_000 // microseconds between frames
	go func() {
		for i := 0; i < 40; i++ {
			f := frames[i%len(frames)]
			// uint32 arithmetic wraps here exactly as the camera's does: the
			// fourth step crosses zero.
			f.Micros = uint32(0xffffffff-3*step) + uint32(i)*step
			st.Frame(f)
			time.Sleep(20 * time.Millisecond)
		}
	}()

	select {
	case <-enough:
	case <-time.After(15 * time.Second):
		t.Fatal("not enough RTP packets reached the client")
	}

	mu.Lock()
	defer mu.Unlock()
	for i := 1; i < len(stamps); i++ {
		// The RTP clock itself won't wrap within this test.
		if stamps[i] < stamps[i-1] {
			t.Fatalf("RTP timestamp went backwards at packet %d: %d -> %d (a drop of %d ticks, %.0fs)",
				i, stamps[i-1], stamps[i],
				stamps[i-1]-stamps[i], float64(stamps[i-1]-stamps[i])/rtpClockHz)
		}
	}
	t.Logf("%d packets, timestamps %d..%d, monotonic across the wrap",
		len(stamps), stamps[0], stamps[len(stamps)-1])
}
