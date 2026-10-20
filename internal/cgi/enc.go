package cgi

import (
	"context"
	"encoding/json"
	"fmt"
)

// EncStream is one stream's encoder settings. Size is the camera's string
// ("4096*1152"); Width and Height must agree with it. GOP is the camera's
// "interframe space" setting, a small number (1-2), not frames per GOP.
type EncStream struct {
	BitRate   int    `json:"bitRate"`
	FrameRate int    `json:"frameRate"`
	GOP       int    `json:"gop"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Size      string `json:"size"`
	Profile   string `json:"profile"`
	VType     string `json:"vType"`
}

// Enc is a channel's encoder settings: one block per stream, plus whether
// audio is encoded at all.
type Enc struct {
	Audio  int       `json:"audio"`
	Main   EncStream `json:"mainStream"`
	Sub    EncStream `json:"subStream"`
	Extern EncStream `json:"extStream"`
}

// EncStreamOptions is what one stream accepts at a given Size. BitRates,
// FrameRates, Profiles and VTypes are exact lists; other values are refused.
type EncStreamOptions struct {
	Size       string
	Width      int
	Height     int
	BitRates   []int
	FrameRates []int
	GOPMin     int
	GOPMax     int
	Profiles   []string
	VTypes     []string

	// Default is the camera's default for this size, not the current value.
	DefaultBitRate   int
	DefaultFrameRate int
	DefaultGOP       int
}

// EncOptions is one resolution the camera supports. The camera returns one
// per resolution, and bitrate/frame rate options depend on the resolution.
type EncOptions struct {
	Main EncStreamOptions
	Sub  EncStreamOptions
}

// encStreamRangeJSON is the wire shape of one stream's options.
type encStreamRangeJSON struct {
	BitRate   []int                  `json:"bitRate"`
	FrameRate []int                  `json:"frameRate"`
	Profile   []string               `json:"profile"`
	VType     []string               `json:"vType"`
	Size      string                 `json:"size"`
	Width     int                    `json:"width"`
	Height    int                    `json:"height"`
	GOP       struct{ Min, Max int } `json:"gop"`
	Default   struct {
		BitRate   int `json:"bitRate"`
		FrameRate int `json:"frameRate"`
		GOP       int `json:"gop"`
	} `json:"default"`
}

func (j encStreamRangeJSON) options() EncStreamOptions {
	return EncStreamOptions{
		Size:             j.Size,
		Width:            j.Width,
		Height:           j.Height,
		BitRates:         j.BitRate,
		FrameRates:       j.FrameRate,
		GOPMin:           j.GOP.Min,
		GOPMax:           j.GOP.Max,
		Profiles:         j.Profile,
		VTypes:           j.VType,
		DefaultBitRate:   j.Default.BitRate,
		DefaultFrameRate: j.Default.FrameRate,
		DefaultGOP:       j.Default.GOP,
	}
}

// GetEnc reads the encoder settings and the options the camera accepts.
func (c *Client) GetEnc(ctx context.Context, channel int) (current Enc, options []EncOptions, err error) {
	value, _, rng, err := c.Get(ctx, "GetEnc", channel)
	if err != nil {
		return Enc{}, nil, err
	}

	var cur struct {
		Enc Enc `json:"Enc"`
	}
	if err := json.Unmarshal(value, &cur); err != nil {
		return Enc{}, nil, fmt.Errorf("cgi: parse GetEnc: %w", err)
	}

	// Usually an array; accept a single object too.
	var asArray struct {
		Enc []struct {
			Main encStreamRangeJSON `json:"mainStream"`
			Sub  encStreamRangeJSON `json:"subStream"`
		} `json:"Enc"`
	}
	if err := json.Unmarshal(rng, &asArray); err == nil && len(asArray.Enc) > 0 {
		for _, e := range asArray.Enc {
			options = append(options, EncOptions{Main: e.Main.options(), Sub: e.Sub.options()})
		}
		return cur.Enc, options, nil
	}

	var asObject struct {
		Enc struct {
			Main encStreamRangeJSON `json:"mainStream"`
			Sub  encStreamRangeJSON `json:"subStream"`
		} `json:"Enc"`
	}
	if err := json.Unmarshal(rng, &asObject); err == nil && asObject.Enc.Main.Size != "" {
		options = append(options, EncOptions{
			Main: asObject.Enc.Main.options(),
			Sub:  asObject.Enc.Sub.options(),
		})
	}
	return cur.Enc, options, nil
}

// SetEnc writes the full encoder block. Streams restart with the new
// parameters, so readers may need to reconnect.
func (c *Client) SetEnc(ctx context.Context, channel int, e Enc) error {
	stream := func(s EncStream) map[string]any {
		return map[string]any{
			"bitRate":   s.BitRate,
			"frameRate": s.FrameRate,
			"gop":       s.GOP,
			"width":     s.Width,
			"height":    s.Height,
			"size":      s.Size,
			"profile":   s.Profile,
			"vType":     s.VType,
		}
	}
	return c.Set(ctx, "SetEnc", map[string]any{
		"Enc": map[string]any{
			"channel":    channel,
			"audio":      e.Audio,
			"mainStream": stream(e.Main),
			"subStream":  stream(e.Sub),
		},
	})
}
