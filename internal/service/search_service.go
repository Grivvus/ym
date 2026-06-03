package service

import (
	"context"
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
	return repository.SearchResult{}, fmt.Errorf("not implemented")
}
