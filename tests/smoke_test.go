package tests

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/Grivvus/ym/internal/storage"
)

func (s *IntegrationTestSuite) TestEnvironmentBootstrapsDatabaseAndStorage() {
	env := s.env

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var exists bool
	err := env.DB.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'user'
		)`,
	).Scan(&exists)
	s.Require().NoError(err)
	s.True(exists)

	const objectID = "integration-smoke-track"

	payload := []byte("integration-smoke-payload")
	s.Require().NoError(env.Storage.PutTrack(
		ctx, objectID, bytes.NewReader(payload),
		storage.PutTrackOptions{Size: int64(len(payload))},
	))
	s.T().Cleanup(func() {
		s.Require().NoError(env.Storage.RemoveTrack(context.Background(), objectID))
	})

	reader, err := env.Storage.GetTrack(ctx, objectID)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(reader.Close())
	})

	actual, err := io.ReadAll(reader)
	s.Require().NoError(err)
	s.Equal(payload, actual)
}
