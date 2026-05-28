package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Grivvus/ym/internal/api"
)

func (s *IntegrationTestSuite) TestDownloadTrackHeadReturnsDownloadMetadata() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userResp := s.registerUser(api.UserAuth{
		Username: "download-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, userResp.StatusCode)

	trackID, checksum := s.createDownloadableTrack(ctx, userResp.Body.UserId)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodHead,
		fmt.Sprintf("%s/tracks/%d/download?quality=standard", s.server.URL, trackID),
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(resp.Body.Close())
	}()

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("audio/ogg", resp.Header.Get("Content-Type"))
	s.Equal("bytes", resp.Header.Get("Accept-Ranges"))
	s.Equal("standard", resp.Header.Get("X-Track-Quality-Requested"))
	s.Equal("fast", resp.Header.Get("X-Track-Quality-Resolved"))
	s.Equal(checksum, resp.Header.Get("X-Track-Checksum-Sha256"))
	s.NotEmpty(resp.Header.Get("ETag"))
	s.True(
		strings.Contains(
			resp.Header.Get("Content-Disposition"),
			fmt.Sprintf("filename=track-%d-fast.opus", trackID),
		),
	)
}

func (s *IntegrationTestSuite) TestDownloadTrackReturnsFileAndHeaders() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userResp := s.registerUser(api.UserAuth{
		Username: "download-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, userResp.StatusCode)

	trackID, checksum := s.createDownloadableTrack(ctx, userResp.Body.UserId)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/tracks/%d/download?quality=fast", s.server.URL, trackID),
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal([]byte("fast-track-payload"), body)
	s.Equal("fast", resp.Header.Get("X-Track-Quality-Resolved"))
	s.Equal(checksum, resp.Header.Get("X-Track-Checksum-Sha256"))
	s.True(
		strings.Contains(
			resp.Header.Get("Content-Disposition"),
			fmt.Sprintf("filename=track-%d-fast.opus", trackID),
		),
	)
}

func (s *IntegrationTestSuite) TestDownloadTrackForbidsPrivateTrackForAnotherUser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "download-owner",
		Password: "password-1",
	})
	otherResp := s.registerUser(api.UserAuth{
		Username: "download-other",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, otherResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)

	statusCode, respBody := s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/tracks/%d/download?quality=fast", trackID),
		nil,
		otherResp.Body.AccessToken,
	)

	var errorResp api.ErrorResponse
	s.Require().NoError(json.Unmarshal(respBody, &errorResp))
	s.Equal(http.StatusForbidden, statusCode)
	s.Contains(errorResp.Error, "user can't have access to this track")
}
