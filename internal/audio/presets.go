package audio

import "fmt"

type Preset int

const (
	PresetFast Preset = iota + 1
	PresetStandard
	PresetHigh
	PresetLossless
)

func PresetFromString(s string) (Preset, error) {
	switch s {
	case "fast":
		return PresetFast, nil
	case "standard":
		return PresetStandard, nil
	case "high":
		return PresetHigh, nil
	case "lossless":
		return PresetLossless, nil
	default:
		return Preset(0), fmt.Errorf("this preset didn't match to any of existing one")
	}
}

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
		return "unknown preset"
	}
}
