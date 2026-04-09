package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
)

type ArtworkOwner struct {
	Kind string
	ID   int32
	Name string
}

func (owner ArtworkOwner) Key() string {
	return fmt.Sprintf("%v_%v_%v", owner.Kind, owner.ID, owner.Name)
}

type ArtworkLoader func(ctx context.Context, id int32) (ArtworkOwner, error)

type ArtworkManager struct {
	storage storage.Storage
	load    ArtworkLoader
	logger  *slog.Logger
}

func NewArtworkManager(
	storage storage.Storage, load ArtworkLoader, logger *slog.Logger,
) ArtworkManager {
	return ArtworkManager{
		storage: storage,
		load:    load,
		logger:  logger,
	}
}

func (m ArtworkManager) Upload(ctx context.Context, id int32, src io.Reader) error {
	owner, err := m.load(ctx, id)
	if err != nil {
		return err
	}
	rc, err := transcoder.ToWebp(src)
	if err != nil {
		return fmt.Errorf(
			"%w: image may be damaged or format is unkown, cause: %w",
			ErrBadParams, err,
		)
	}
	defer func() { _ = rc.Close() }()
	err = m.storage.PutImage(ctx, owner.Key(), rc)
	if err != nil {
		if errors.Is(err, storage.ErrBadObject) {
			return fmt.Errorf("%w file is too large or too small: %w", ErrBadParams, err)
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}

func (m ArtworkManager) Get(ctx context.Context, id int32) ([]byte, error) {
	owner, err := m.load(ctx, id)
	if err != nil {
		return nil, err
	}
	img, err := m.storage.GetImage(ctx, owner.Key())
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, NewErrNotFound(owner.Kind, owner.ID)
		}
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return img, nil
}

func (m ArtworkManager) Delete(ctx context.Context, id int32) error {
	owner, err := m.load(ctx, id)
	if err != nil {
		return err
	}
	err = m.storage.RemoveImage(ctx, owner.Key())
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			// nop
			return nil
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}
