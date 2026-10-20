package cgi

import (
	"context"
	"encoding/json"
	"fmt"
)

// Speaker volume (GetAudioCfg) and alarm sound on/off (GetAudioAlarmV20).
// The sound list, upload and siren commands return -9 (not supported) on the
// models tested, so they aren't implemented.

// AudioCfg is the camera's speaker volume.
type AudioCfg struct {
	Volume int `json:"volume"`
}

// VolumeRange is the volume range the camera reports.
type VolumeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// GetAudioCfg reads the speaker volume and the range it may take.
func (c *Client) GetAudioCfg(ctx context.Context, channel int) (AudioCfg, VolumeRange, error) {
	value, _, rng, err := c.Get(ctx, "GetAudioCfg", channel)
	if err != nil {
		return AudioCfg{}, VolumeRange{}, err
	}
	var v struct {
		AudioCfg AudioCfg `json:"AudioCfg"`
	}
	if err := json.Unmarshal(value, &v); err != nil {
		return AudioCfg{}, VolumeRange{}, fmt.Errorf("cgi: parse audio cfg: %w", err)
	}
	var r struct {
		AudioCfg struct {
			Volume VolumeRange `json:"volume"`
		} `json:"AudioCfg"`
	}
	// A missing range is not an error; the caller gets a zero range.
	_ = json.Unmarshal(rng, &r)
	return v.AudioCfg, r.AudioCfg.Volume, nil
}

// SetAudioCfg writes the speaker volume.
func (c *Client) SetAudioCfg(ctx context.Context, channel, volume int) error {
	return c.Set(ctx, "SetAudioCfg", map[string]any{
		"AudioCfg": map[string]any{"channel": channel, "volume": volume},
	})
}

// GetAudioAlarm reads the alarm sound document as a map, since the schedule
// entries differ by model. Callers change a field and write it back whole.
func (c *Client) GetAudioAlarm(ctx context.Context, channel int) (map[string]any, error) {
	value, _, _, err := c.Get(ctx, "GetAudioAlarmV20", channel)
	if err != nil {
		return nil, err
	}
	var v struct {
		Audio map[string]any `json:"Audio"`
	}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, fmt.Errorf("cgi: parse audio alarm: %w", err)
	}
	if v.Audio == nil {
		return nil, fmt.Errorf("cgi: GetAudioAlarmV20 carried no Audio document")
	}
	return v.Audio, nil
}

// AudioAlarmEnabled reads the on/off out of that document.
func AudioAlarmEnabled(doc map[string]any) bool {
	f, ok := doc["enable"].(float64)
	return ok && f != 0
}

// SetAudioAlarmEnabled turns the alarm sound on or off. It writes back the
// camera's document with only that field changed, keeping the schedule.
func (c *Client) SetAudioAlarmEnabled(ctx context.Context, channel int, doc map[string]any, on bool) error {
	next := make(map[string]any, len(doc)+1)
	for k, v := range doc {
		next[k] = v
	}
	if on {
		next["enable"] = 1
	} else {
		next["enable"] = 0
	}
	return c.Set(ctx, "SetAudioAlarmV20", map[string]any{"Audio": next})
}
