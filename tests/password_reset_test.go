package tests

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
)

func (s *IntegrationTestSuite) TestPasswordReset_RequestReturnsAcceptedForUnknownEmail() {
	resp := s.requestPasswordReset("unknown@example.com")

	s.Equal(http.StatusAccepted, resp.StatusCode)
	s.Equal(
		"if an account with that email exists, a reset code has been sent",
		resp.Body.Msg,
	)
	_, exists := s.resetMailer.LastCode("unknown@example.com")
	s.False(exists)
}

func (s *IntegrationTestSuite) TestPasswordReset_ConfirmChangesPasswordAndInvalidatesRefreshToken() {
	registerResp := s.registerUser(api.UserAuth{
		Username: "reset-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, registerResp.StatusCode)
	s.setUserEmail(registerResp.Body.UserId, "reset@example.com", "reset-user")

	requestResp := s.requestPasswordReset("reset@example.com")
	s.Equal(http.StatusAccepted, requestResp.StatusCode)

	code, exists := s.resetMailer.LastCode("reset@example.com")
	s.True(exists)
	s.Len(code, 6)

	oldAccessToken := registerResp.Body.AccessToken
	oldRefreshToken := registerResp.Body.RefreshToken

	confirmResp := s.confirmPasswordReset("reset@example.com", code, "password-2")
	s.Equal(http.StatusOK, confirmResp.StatusCode)
	s.Equal("password was successfully reset", confirmResp.Body.Msg)

	refreshResp := s.refreshTokens(oldRefreshToken)
	s.Equal(http.StatusUnauthorized, refreshResp.StatusCode)
	s.Equal("invalid refresh token", refreshResp.Error.Error)

	loginOldPasswordResp := s.loginUser(api.UserAuth{
		Username: "reset-user",
		Password: "password-1",
	})
	s.Equal(http.StatusUnauthorized, loginOldPasswordResp.StatusCode)
	s.Equal("invalid credentials", loginOldPasswordResp.Error.Error)

	loginNewPasswordResp := s.loginUser(api.UserAuth{
		Username: "reset-user",
		Password: "password-2",
	})
	s.Equal(http.StatusOK, loginNewPasswordResp.StatusCode)
	s.NotEmpty(loginNewPasswordResp.Body.AccessToken)

	statusCode, body := s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/users/%d", registerResp.Body.UserId),
		nil,
		oldAccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	var user api.UserReturn
	s.Require().NoError(json.Unmarshal(body, &user))
	s.Equal("reset@example.com", *user.Email)
}
