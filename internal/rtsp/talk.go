package rtsp

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/g711"
)

// Talker opens a two-way audio session on the camera behind a stream.
type Talker interface {
	Open(ctx context.Context) (TalkSession, error)
}

// TalkSession is one open talk session. Write takes 16 bit mono PCM at rate.
type TalkSession interface {
	Write(rate int, pcm []int16) error
	Ping() error
	Close() error
}

// backChannelMedia is what a stream offers a client that asks for an ONVIF
// back channel: G.711 at 8 kHz, in both laws. That is what go2rtc, and so
// Frigate's talk button, sends without transcoding, since browsers speak it
// natively over WebRTC.
func backChannelMedia() *description.Media {
	return &description.Media{
		Type:          description.MediaTypeAudio,
		IsBackChannel: true,
		Formats: []format.Format{
			&format.G711{PayloadTyp: 0, MULaw: true, SampleRate: 8000, ChannelCount: 1},
			&format.G711{PayloadTyp: 8, MULaw: false, SampleRate: 8000, ChannelCount: 1},
		},
	}
}

// decodeG711 turns one RTP payload into PCM.
func decodeG711(f *format.G711, payload []byte) []int16 {
	var be []byte
	if f.MULaw {
		var m g711.Mulaw
		m.Unmarshal(payload)
		be = m
	} else {
		var a g711.Alaw
		a.Unmarshal(payload)
		be = a
	}
	out := make([]int16, len(be)/2)
	for i := range out {
		out[i] = int16(binary.BigEndian.Uint16(be[i*2:]))
	}
	return out
}

// talkIdle is how long a back channel may go quiet before its camera
// session is released. go2rtc keeps the RTSP session up for as long as
// anyone is watching, but only sends audio while someone holds the talk
// button, and a camera mid talk session is a camera nobody else can talk to.
const talkIdle = 3 * time.Second

// talkPing is how often an open session is pinged while waiting for more
// audio, well inside the connection's idle timeout.
const talkPing = 5 * time.Second

// backChannel carries one RTSP session's back channel audio to the camera.
// Packets arrive on gortsplib's goroutine, which must not block, so they are
// handed over a buffered channel to a goroutine of its own that owns the
// camera session: opening one takes a login, about a second, and the buffer
// holds the speech that arrives meanwhile.
type backChannel struct {
	name   string
	talker Talker
	audio  chan []int16
	stop   chan struct{}
	once   sync.Once
	done   chan struct{}
}

func newBackChannel(name string, t Talker) *backChannel {
	b := &backChannel{
		name:   name,
		talker: t,
		// 128 packets of 20 ms is about two and a half seconds.
		audio: make(chan []int16, 128),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go b.run()
	return b
}

// packet takes one RTP payload. It never blocks; a backlog drops audio.
func (b *backChannel) packet(f *format.G711, payload []byte) {
	select {
	case b.audio <- decodeG711(f, payload):
	default:
	}
}

func (b *backChannel) close() {
	b.once.Do(func() { close(b.stop) })
	<-b.done
}

func (b *backChannel) run() {
	defer close(b.done)
	var sess TalkSession
	release := func() {
		if sess != nil {
			if err := sess.Close(); err != nil {
				log.Printf("rtsp: %s: talk: close: %v", b.name, err)
			}
			sess = nil
		}
	}
	defer release()

	// After a failed open, wait before trying again rather than redialling
	// the camera for every 20 ms packet of a held button.
	var retryAt time.Time
	idle := time.NewTimer(talkIdle)
	defer idle.Stop()
	ping := time.NewTicker(talkPing)
	defer ping.Stop()

	for {
		select {
		case <-b.stop:
			return
		case pcm := <-b.audio:
			if sess == nil {
				if time.Now().Before(retryAt) {
					continue
				}
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					select {
					case <-b.stop:
						cancel()
					case <-ctx.Done():
					}
				}()
				s, err := b.talker.Open(ctx)
				cancel()
				if err != nil {
					log.Printf("rtsp: %s: talk: %v", b.name, err)
					retryAt = time.Now().Add(10 * time.Second)
					continue
				}
				log.Printf("rtsp: %s: talk started", b.name)
				sess = s
			}
			if err := sess.Write(8000, pcm); err != nil {
				log.Printf("rtsp: %s: talk: %v", b.name, err)
				release()
				continue
			}
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(talkIdle)
		case <-idle.C:
			if sess != nil {
				log.Printf("rtsp: %s: talk ended", b.name)
			}
			release()
			idle.Reset(talkIdle)
		case <-ping.C:
			if sess != nil {
				if err := sess.Ping(); err != nil && !errors.Is(err, context.Canceled) {
					release()
				}
			}
		}
	}
}
