package tests

import (
	"net/http"

	"github.com/Grivvus/ym/internal/api"
)

func (s *IntegrationTestSuite) TestRegister_FirstUserBecomesSuperuser() {
	resp := s.registerUser(api.UserAuth{
		Username: "first-user",
		Password: "password-1",
	})

	s.Equal(http.StatusCreated, resp.StatusCode)
	s.NotZero(resp.Body.UserId)
	s.NotEmpty(resp.Body.AccessToken)
	s.NotEmpty(resp.Body.RefreshToken)
	s.Equal("bearer", resp.Body.TokenType)
	s.True(s.userIsSuperuser(resp.Body.UserId))
}

func (s *IntegrationTestSuite) TestRegister_SubsequentUserDoesNotBecomeSuperuser() {
	firstResp := s.registerUser(api.UserAuth{
		Username: "first-user",
		Password: "password-1",
	})
	secondResp := s.registerUser(api.UserAuth{
		Username: "second-user",
		Password: "password-2",
	})

	s.Equal(http.StatusCreated, firstResp.StatusCode)
	s.Equal(http.StatusCreated, secondResp.StatusCode)
	s.True(s.userIsSuperuser(firstResp.Body.UserId))
	s.False(s.userIsSuperuser(secondResp.Body.UserId))
}
