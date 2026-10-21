package rtsp

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/g711"
)

// fakeTalker stands in for a camera's speaker and records what reached it.
type fakeTalker struct {
	mu      sync.Mutex
	opens   int
	closes  int
	samples []int16
	rates   map[int]bool
}

func (f *fakeTalker) Open(context.Context) (TalkSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opens++
	return fakeSession{f}, nil
}

type fakeSession struct{ f *fakeTalker }

func (s fakeSession) Write(rate int, pcm []int16) error {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	if s.f.rates == nil {
		s.f.rates = map[int]bool{}
	}
	s.f.rates[rate] = true
	s.f.samples = append(s.f.samples, pcm...)
	return nil
}
func (s fakeSession) Ping() error { return nil }
func (s fakeSession) Close() error {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.closes++
	return nil
}

func (f *fakeTalker) snapshot() (opens, closes, n int, rates map[int]bool, first int16) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.samples) > 0 {
		first = f.samples[0]
	}
	return f.opens, f.closes, len(f.samples), f.rates, first
}

func backChannelIn(desc *description.Session) (*description.Media, *format.G711) {
	for _, m := range desc.Medias {
		if !m.IsBackChannel {
			continue
		}
		for _, f := range m.Formats {
			if g, ok := f.(*format.G711); ok {
				return m, g
			}
		}
	}
	return nil, nil
}

func describeWith(t *testing.T, addr, path string, back bool) (*gortsplib.Client, *description.Session) {
	t.Helper()
	u, err := base.ParseURL("rtsp://" + addr + "/" + path)
	if err != nil {
		t.Fatal(err)
	}
	tr := gortsplib.ProtocolTCP
	c := &gortsplib.Client{Scheme: u.Scheme, Host: u.Host, Protocol: &tr, RequestBackChannels: back}
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	desc, _, err := c.Describe(u)
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	return c, desc
}

// The whole feature, from the client's side: a go2rtc-style client asks for
// the back channel, finds G.711 there, sends speech, and it reaches the
// camera's speaker as PCM. Hanging up releases the camera.
func TestBackChannelAudioReachesTheCamera(t *testing.T) {
	frames := captureFrames(t)
	srv, st := serverWithStream(t, "cam")
	ft := &fakeTalker{}
	st.SetTalker(ft)
	st.SetSpeaker(true)
	feedUntilReady(t, st, frames)

	c, desc := describeWith(t, srv.Addr(), "cam", true)
	medi, g := backChannelIn(desc)
	if medi == nil {
		t.Fatal("no G.711 back channel offered to a client that asked for one")
	}
	if err := c.SetupAll(desc.BaseURL, desc.Medias); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := c.Play(nil); err != nil {
		t.Fatalf("play: %v", err)
	}

	enc, err := g.CreateEncoder()
	if err != nil {
		t.Fatal(err)
	}
	// 20 ms packets of a constant level, so what arrives can be checked.
	pcm := make([]byte, 320)
	for i := 0; i < len(pcm); i += 2 {
		pcm[i], pcm[i+1] = 0x10, 0x00 // 4096, big endian
	}
	var payload []byte
	if g.MULaw {
		payload, _ = g711.Mulaw(pcm).Marshal()
	} else {
		payload, _ = g711.Alaw(pcm).Marshal()
	}
	for i := 0; i < 25; i++ {
		pkts, err := enc.Encode(payload)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range pkts {
			p.Timestamp = uint32(i * 160)
			if err := c.WritePacketRTP(medi, p); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
		time.Sleep(5 * time.Millisecond)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, n, _, _ := ft.snapshot(); n >= 25*160 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	opens, _, n, rates, first := ft.snapshot()
	if opens != 1 || n < 25*160 {
		t.Fatalf("camera saw %d opens and %d samples, want 1 and %d", opens, n, 25*160)
	}
	if !rates[8000] || len(rates) != 1 {
		t.Errorf("rates %v, want only 8000", rates)
	}
	if first < 3900 || first > 4300 {
		t.Errorf("first sample %d, want about 4096 after G.711", first)
	}

	c.Close()
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, closes, _, _, _ := ft.snapshot(); closes == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("hanging up did not release the camera's talk session")
}

// A camera with no speaker offers no back channel, so Frigate shows no talk
// button for it, and nothing can ever send it a talk configuration.
func TestNoBackChannelWithoutASpeaker(t *testing.T) {
	frames := captureFrames(t)
	srv, st := serverWithStream(t, "cam")
	st.SetTalker(&fakeTalker{})
	st.SetSpeaker(false)
	feedUntilReady(t, st, frames)

	if _, desc := describeWith(t, srv.Addr(), "cam", true); func() bool { m, _ := backChannelIn(desc); return m != nil }() {
		t.Fatal("back channel offered for a camera with no speaker")
	}
}

// A plain player, ffmpeg included, never asks for a back channel and must
// not be handed one: an unexpected sendonly track is how a recorder that
// works today breaks after an upgrade.
func TestPlainClientsSeeNoBackChannel(t *testing.T) {
	frames := captureFrames(t)
	srv, st := serverWithStream(t, "cam")
	st.SetTalker(&fakeTalker{})
	st.SetSpeaker(true)
	feedUntilReady(t, st, frames)

	_, desc := describeWith(t, srv.Addr(), "cam", false)
	for _, m := range desc.Medias {
		if m.IsBackChannel {
			t.Fatal("back channel offered to a client that did not ask for one")
		}
	}
}
