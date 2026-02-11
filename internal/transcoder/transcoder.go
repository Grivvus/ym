package transcoder

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Preset int

const (
	PresetFast Preset = iota + 1
	PresetStandard
	PresetHigh
	PresetLossless
)

func (p Preset) String() string {
	switch p {
	case PresetFast:
		return "fast"
	case PresetStandard:
		return "standard"
	case PresetHigh:
		return "high"
	case PresetLossless:
		return "lossless"
	default:
		return "unkown preset"
	}
}

const TimeoutPerPreset = 5 * time.Second

var presetArgs = map[Preset][]string{
	PresetFast:     {"-vn", "-c:a", "libopus", "-b:a", "48k", "-vbr", "on", "-ac", "2", "-ar", "48000"},
	PresetStandard: {"-vn", "-c:a", "libopus", "-b:a", "112k", "-vbr", "on", "-ac", "2", "-ar", "48000"},
	PresetHigh:     {"-vn", "-c:a", "aac", "-b:a", "256k", "-ar", "44100", "-ac", "2", "-movflags", "+faststart"},
	PresetLossless: {"-vn", "-c:a", "flac", "-compression_level", "5"},
}

func Transcode(ctx context.Context, fname string) (map[Preset]string, error) {
	presets := []Preset{PresetFast, PresetStandard, PresetHigh, PresetLossless}
	transcodedFiles := make(map[Preset]string, 4)
	for _, p := range presets {
		newName := transcodedName(fname, p)
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
			// if we're exiting with error, we should remove all transcoded files
			// or we could create some worker, that remove all files,
			// that are older than some time, 30 mins for example
			panic("not implemented")
		}
		if pctx.Err() != nil {
			return nil, pctx.Err()
		}
		transcodedFiles[p] = newName
	}
	return transcodedFiles, nil
}

func TranscodeConcurrent(ctx context.Context, fname string) (map[Preset]string, error) {
	presets := []Preset{PresetFast, PresetStandard, PresetHigh, PresetLossless}
	transcodedFiles := make(map[Preset]string, 4)
	c := make(chan Preset)
	pctx, cancel := context.WithTimeout(ctx, TimeoutPerPreset)
	defer cancel()
	for _, p := range presets {
		currentPreset := p
		go func() {
			newName := transcodedName(fname, currentPreset)
			allArgs := []string{"-y", "-i", fname}
			allArgs = append(allArgs, presetArgs[currentPreset]...)
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
			c <- currentPreset
		}()
	}
	done := 0
	for {
		select {
		case p := <-c:
			transcodedFiles[p] = transcodedName(fname, p)
			done++
			if done == len(presets) {
				return transcodedFiles, nil
			}
		case <-pctx.Done():
			return nil, fmt.Errorf("deadline on operation")
		}
	}
}

func transcodedName(fname string, preset Preset) string {
	var b strings.Builder
	if !strings.Contains(fname, ".") {
		b.WriteString(fname)
	} else {
		splited := strings.Split(fname, ".")
		for i, part := range splited {
			if i == len(splited)-1 {
				continue
			}
			b.WriteString(part)
		}
	}
	b.WriteByte('_')
	b.WriteString(preset.String())
	return b.String()
}
