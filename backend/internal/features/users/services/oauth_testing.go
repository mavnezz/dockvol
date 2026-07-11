package users_services

import (
	"golang.org/x/oauth2"

	users_dto "dockvol-backend/internal/features/users/dto"
)

func (s *UserService) HandleGitHubOAuthWithMockEndpoint(
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGitHubOAuthWithEndpoint(code, redirectUri, endpoint, userAPIURL)
}

func (s *UserService) HandleGoogleOAuthWithMockEndpoint(
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGoogleOAuthWithEndpoint(code, redirectUri, endpoint, userAPIURL)
}
