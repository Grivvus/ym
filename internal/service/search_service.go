package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Grivvus/ym/internal/repository"
)

type SearchService struct {
	logger *slog.Logger
	repo   repository.SearchRepository
}

func NewSearchService(logger *slog.Logger, searchRepo repository.SearchRepository) SearchService {
	return SearchService{
		logger: logger,
		repo:   searchRepo,
	}
}

func (s *SearchService) Search(
	ctx context.Context, params repository.SearchParams,
) (repository.SearchResult, error) {
	result, err := s.repo.Search(ctx, params)
	if err != nil {
		if !errors.Is(err, repository.ErrUnknownDBError) {
			s.logger.Warn("unexpected state, error should always be UnknownDBError")
		}
		return repository.SearchResult{}, fmt.Errorf("%w: %w", ErrUnknownDBError, err)
	}
	return result, nil
}
