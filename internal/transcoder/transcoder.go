package transcoder

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type Preset int

const (
	PresetFast Preset = iota + 1
	PresetStandard
	PresetHigh
	PresetLossless
)

const TimeoutPerPreset = 5 * time.Second

var presetArgs = map[Preset][]string{
	PresetFast:     {"-vn", "-c:a", "libopus", "-b:a", "48k", "-vbr", "on", "-ac", "2", "-ar", "48000"},
	PresetStandard: {"-vn", "-c:a", "libopus", "-b:a", "112k", "-vbr", "on", "-ac", "2", "-ar", "48000"},
	PresetHigh:     {"-vn", "-c:a", "aac", "-b:a", "256k", "-ar", "44100", "-ac", "2", "-movflags", "+faststart"},
	PresetLossless: {"-vn", "-c:a", "flac", "-compression_level", "5"},
}

func Transcode(ctx context.Context, fname string) (map[Preset]string, error) {
	presets := []Preset{PresetFast, PresetStandard, PresetHigh, PresetLossless}
	for _, p := range presets {
		newName := transocdedName(fname, p)
		allArgs := []string{"-y", "-i", fname}
		allArgs = append(allArgs, presetArgs[p]...)
		allArgs = append(allArgs, newName)
		pctx, cancel := context.WithTimeout(context.Background(), TimeoutPerPreset)
		defer cancel()

		cmd := exec.CommandContext(pctx, "ffmpeg", allArgs...)
		var buf bytes.Buffer
		cmd.Stderr = &buf
		cmd.Stdout = &buf

		if err := cmd.Run(); err != nil {
			panic("not implemented")
		}
	}
}

func transocdedName(fname string, preset Preset) string {}
