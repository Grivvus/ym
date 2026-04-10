package transcoder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Grivvus/ym/internal/audio"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/storage"
)

var presetArgs = map[audio.Preset][]string{
	audio.PresetFast:     {"-vn", "-c:a", "libopus", "-b:a", "48k", "-vbr", "on", "-ac", "2", "-ar", "48000", "-f", "opus"},
	audio.PresetStandard: {"-vn", "-c:a", "libopus", "-b:a", "112k", "-vbr", "on", "-ac", "2", "-ar", "48000", "-f", "opus"},
	audio.PresetHigh:     {"-vn", "-c:a", "aac", "-b:a", "256k", "-ar", "44100", "-ac", "2", "-movflags", "+faststart", "-f", "mov"},
	audio.PresetLossless: {"-vn", "-c:a", "flac", "-compression_level", "5", "-f", "flac"},
}

const perPresetTimeout = 60 * time.Second

type Transcoder struct {
	logger            *slog.Logger
	storage           storage.Storage
	repo              *repository.TranscodingQueueRepository
	transcodingSignal <-chan struct{}
	isWorkerStarted   atomic.Bool
	tickerChan        <-chan time.Time
}

func NewTranscoder(
	logger *slog.Logger,
	storage storage.Storage,
	repo *repository.TranscodingQueueRepository,
	transcodingQueue <-chan struct{},
) *Transcoder {
	return &Transcoder{
		logger:            logger,
		storage:           storage,
		repo:              repo,
		transcodingSignal: transcodingQueue,
		isWorkerStarted:   atomic.Bool{},
		tickerChan:        time.Tick(5 * time.Second),
	}
}

func (t *Transcoder) StartListener(ctx context.Context) {
	go func() {
		for {
			select {
			case <-t.transcodingSignal:
				t.checkAndLaunchWorker(ctx)
			case <-t.tickerChan:
				t.checkAndLaunchWorker(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (t *Transcoder) checkAndLaunchWorker(ctx context.Context) {
	if t.isWorkerStarted.CompareAndSwap(false, true) {
		go t.startWorker(ctx)
	}
}

func (t *Transcoder) startWorker(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		queue, errs := t.repo.GetTranscodingQueue(ctx)
		go func() {
			for err := range errs {
				t.logger.Error("failed to get queue", "err", err.Error())
			}
		}()
		for transcodingInfo := range queue {
			t.job(ctx, transcodingInfo)
		}
	})
	wg.Wait()
	t.isWorkerStarted.Store(false)
}

func (t *Transcoder) job(ctx context.Context, transcodingInfo db.GetTranscodingQueueRow) {
	presets := []audio.Preset{
		audio.PresetFast, audio.PresetStandard, audio.PresetHigh,
	}
	presetsToName := make(map[audio.Preset]string)
	for _, p := range presets {
		presetsToName[p] = TranscodedName(transcodingInfo.TrackOriginalFileName, p)
	}
	err := t.writeTrackToTmpFile(ctx, transcodingInfo)
	defer func() {
		go t.removeTmpFiles(transcodingInfo.TrackOriginalFileName)
	}()
	if err != nil {
		t.logger.Error("failed to write track to tmp file", "err", err.Error())
		return
	}
	transcodingCtx, cancel := context.WithTimeout(ctx, perPresetTimeout)
	defer cancel()
	err = t.transcodeConcurrent(transcodingCtx, transcodingInfo.TrackOriginalFileName, presets)
	if err != nil {
		t.logger.Error("can't transcode track", "error", err)
		_ = t.repo.OnFailedTranscoding(ctx, transcodingInfo.ID, err)
		return
	}
	t.logger.Info(
		"track transcoded successfully",
		"track", transcodingInfo.TrackOriginalFileName,
	)
	duration, err := t.probeDurationMs(ctx, transcodingInfo.TrackOriginalFileName)
	if err != nil {
		t.logger.Error("can't probe duration", "error", err)
		return
	}
	err = t.uploadTmpFilesToStorage(ctx, presetsToName)
	if err != nil {
		t.logger.Error("can't upload tmp files", "error", err)
		return
	}
	err = t.repo.RemoveFromQueueAndUpdateTrack(
		ctx, transcodingInfo.ID, duration, presetsToName,
	)
	if err != nil {
		t.logger.Error(
			"can't update db after successful transcoding",
			"error", err,
		)
	}
}

func (t *Transcoder) uploadTmpFilesToStorage(
	ctx context.Context, presetsToName map[audio.Preset]string,
) error {
	for _, presetFname := range presetsToName {
		f, err := os.Open(presetFname)
		if err != nil {
			t.logger.Error("can't open transcoded file", "err", err.Error())
			return err
		}
		fstat, err := f.Stat()
		if err != nil {
			t.logger.Error("can't stat transcoded file", "err", err.Error())
			return err
		}
		defer func() { _ = f.Close() }()
		err = t.storage.PutTrack(
			ctx,
			presetFname,
			f,
			fstat.Size(),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Transcoder) writeTrackToTmpFile(
	ctx context.Context, transcodingInfo db.GetTranscodingQueueRow,
) error {
	trackData, err := t.storage.GetTrack(ctx, transcodingInfo.TrackOriginalFileName)
	if err != nil {
		return err
	}
	defer func() { _ = trackData.Close() }()
	f, err := os.Create(transcodingInfo.TrackOriginalFileName)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, trackData)
	if err != nil {
		return err
	}
	return nil
}

func (t *Transcoder) transcodeConcurrent(
	ctx context.Context, fname string, presets []audio.Preset,
) error {
	transcodedFiles := make(map[audio.Preset]string, 4)
	c := make(chan audio.Preset)

	for _, p := range presets {
		currentPreset := p
		go func() {
			newName := TranscodedName(fname, currentPreset)
			allArgs := []string{"-y", "-i", fname}
			allArgs = append(allArgs, presetArgs[currentPreset]...)
			allArgs = append(allArgs, newName)

			cmd := exec.CommandContext(ctx, "ffmpeg", allArgs...)
			var buf bytes.Buffer
			cmd.Stderr = &buf
			cmd.Stdout = &buf

			if err := cmd.Run(); err != nil {
				t.logger.Error(
					"transcoding ended with an error",
					"buffer output", buf.String(),
				)
			}
			c <- currentPreset
		}()
	}
	done := 0
	for {
		select {
		case p := <-c:
			transcodedFiles[p] = TranscodedName(fname, p)
			done++
			if done == len(presets) {
				return nil
			}
		case <-ctx.Done():
			err := ctx.Err()
			return fmt.Errorf("context was canceled with err: %w", err)
		}
	}
}

func (t *Transcoder) probeDurationMs(ctx context.Context, fname string) (int32, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		fname,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf(
			"ffprobe failed: %w: %s", err, strings.TrimSpace(stderr.String()),
		)
	}

	seconds, err := strconv.ParseFloat(strings.TrimSpace(stdout.String()), 64)
	if err != nil {
		return 0, fmt.Errorf("can't parse duration: %w", err)
	}

	return int32(math.Round(seconds * 1000)), nil
}

func (t *Transcoder) removeTmpFiles(fname string) {
	err := os.Remove(fname)
	if err != nil {
		t.logger.Error("error on removing tmp", "err", err)
	}
	presets := []audio.Preset{
		audio.PresetFast, audio.PresetStandard,
		audio.PresetHigh, audio.PresetLossless,
	}
	for _, p := range presets {
		tmp := TranscodedName(fname, p)
		err := os.Remove(tmp)
		if err != nil {
			t.logger.Error("error on removing tmp", "err", err)
		}
	}
}

func TranscodedName(fname string, preset audio.Preset) string {
	var b strings.Builder
	// skip .extension part if exist
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
