package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tixora/internal/config"
	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

	oauthHTTPTimeout = 10 * time.Second
)

// IAuthService handles Google OAuth login and access/refresh token issuance.
type IAuthService interface {
	HandleGoogleCallback(ctx context.Context, code string) (accessToken, refreshToken string, user *models.User, err error)
	RefreshToken(ctx context.Context, rawRefreshToken string) (accessToken, refreshToken string, user *models.User, err error)
	Logout(ctx context.Context, rawRefreshToken string) error
}

type AuthService struct {
	userRepo         repository.IUserRepository
	refreshTokenRepo repository.IRefreshTokenRepository
	cfg              *config.Config
	httpClient       *http.Client
}

func NewAuthService(userRepo repository.IUserRepository, refreshTokenRepo repository.IRefreshTokenRepository, cfg *config.Config) IAuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		cfg:              cfg,
		httpClient:       &http.Client{Timeout: oauthHTTPTimeout},
	}
}

type googleTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// HandleGoogleCallback exchanges an OAuth authorization code for the caller's
// Google profile, finds or creates the matching local user, and issues a new
// access/refresh token pair for that user.
func (s *AuthService) HandleGoogleCallback(ctx context.Context, code string) (string, string, *models.User, error) {
	if code == "" {
		return "", "", nil, fmt.Errorf("%w: authorization code is required", utils.ErrInvalidInput)
	}

	accessToken, err := s.exchangeCode(ctx, code)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	googleUser, err := s.fetchGoogleUserInfo(ctx, accessToken)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to fetch google user info: %w", err)
	}

	if googleUser.Email == "" || googleUser.ID == "" {
		return "", "", nil, fmt.Errorf("%w: google account is missing required profile fields", utils.ErrInvalidInput)
	}

	user, err := s.userRepo.GetByGoogleID(ctx, googleUser.ID)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to look up user: %w", err)
	}

	if user == nil {
		user = &models.User{
			ID:        utils.GenerateUUID(),
			Email:     googleUser.Email,
			Name:      googleUser.Name,
			GoogleID:  googleUser.ID,
			AvatarURL: googleUser.Picture,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", "", nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else if user.Name != googleUser.Name || user.AvatarURL != googleUser.Picture {
		user.Name = googleUser.Name
		user.AvatarURL = googleUser.Picture
		if err := s.userRepo.Update(ctx, user); err != nil {
			return "", "", nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	jwtAccessToken, rawRefreshToken, _, err := s.issueTokens(ctx, user.ID, user.Email)
	if err != nil {
		return "", "", nil, err
	}

	return jwtAccessToken, rawRefreshToken, user, nil
}

// RefreshToken validates a raw refresh token against the stored hash,
// rotates it (revoking the old one and issuing a new pair), and returns the
// new access/refresh tokens. If a token that was already revoked is
// presented again - a strong signal it was stolen - every refresh token for
// that user is revoked, forcing re-login everywhere.
func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (string, string, *models.User, error) {
	if rawRefreshToken == "" {
		return "", "", nil, fmt.Errorf("%w: refresh token is required", utils.ErrUnauthorized)
	}

	tokenHash := utils.HashToken(rawRefreshToken)
	stored, err := s.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to look up refresh token: %w", err)
	}
	if stored == nil {
		return "", "", nil, fmt.Errorf("%w: invalid refresh token", utils.ErrUnauthorized)
	}

	if stored.RevokedAt != nil {
		if revokeErr := s.refreshTokenRepo.RevokeAllForSubject(ctx, stored.SubjectID, stored.SubjectType); revokeErr != nil {
			return "", "", nil, fmt.Errorf("failed to revoke compromised sessions: %w", revokeErr)
		}
		return "", "", nil, fmt.Errorf("%w: refresh token reuse detected", utils.ErrUnauthorized)
	}

	if stored.ExpiresAt.Before(time.Now()) {
		return "", "", nil, fmt.Errorf("%w: refresh token expired", utils.ErrUnauthorized)
	}

	user, err := s.userRepo.GetByID(ctx, stored.SubjectID)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if user == nil {
		return "", "", nil, fmt.Errorf("%w: user", utils.ErrNotFound)
	}

	newAccessToken, newRawRefreshToken, newRowID, err := s.issueTokens(ctx, user.ID, user.Email)
	if err != nil {
		return "", "", nil, err
	}

	if err := s.refreshTokenRepo.Revoke(ctx, tokenHash, &newRowID); err != nil {
		return "", "", nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	return newAccessToken, newRawRefreshToken, user, nil
}

// Logout revokes the given refresh token so it can no longer be used to
// mint new access tokens. It's idempotent - logging out with an empty or
// already-revoked token is not an error.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	if err := s.refreshTokenRepo.Revoke(ctx, utils.HashToken(rawRefreshToken), nil); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// issueTokens generates a fresh access JWT and a fresh (persisted, hashed)
// refresh token for the given user, returning the new refresh token row's ID
// alongside them so callers can link a rotated-out token to its replacement.
func (s *AuthService) issueTokens(ctx context.Context, userID, email string) (string, string, string, error) {
	accessToken, err := utils.GenerateJWT(userID, email, s.cfg.JWTSecret, s.cfg.JWTAccessExpiry)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	rawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshRow := &models.RefreshToken{
		ID:          utils.GenerateUUID(),
		SubjectID:   userID,
		SubjectType: models.SubjectTypeUser,
		TokenHash:   utils.HashToken(rawRefreshToken),
		ExpiresAt:   time.Now().Add(s.cfg.JWTRefreshExpiry),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshRow); err != nil {
		return "", "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, rawRefreshToken, refreshRow.ID, nil
}

func (s *AuthService) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.cfg.GoogleOAuthID)
	form.Set("client_secret", s.cfg.GoogleOAuthSecret)
	form.Set("redirect_uri", s.cfg.OAuthCallbackURL)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp googleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || tokenResp.Error != "" {
		return "", fmt.Errorf("google token exchange failed: %s %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("google token exchange returned no access token")
	}

	return tokenResp.AccessToken, nil
}

func (s *AuthService) fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo request failed with status %d", resp.StatusCode)
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return &userInfo, nil
}
