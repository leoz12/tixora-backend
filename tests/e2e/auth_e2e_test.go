package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/dto"
)

func TestE2E_UserAuth_MeRequiresAuth(t *testing.T) {
	app := newTestApp(t)

	rec := app.do(t, http.MethodGet, "/api/auth/me", nil, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestE2E_UserAuth_MeReturnsCurrentUser(t *testing.T) {
	app := newTestApp(t)
	app.seedUser(t, "user-1", "jane@example.com")
	cookies := app.userCookies(t, "user-1", "jane@example.com")

	var resp struct {
		Success bool             `json:"success"`
		Data    dto.UserResponse `json:"data"`
	}
	rec := app.do(t, http.MethodGet, "/api/auth/me", nil, &cookies, &resp)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "jane@example.com", resp.Data.Email)
}

func TestE2E_UserAuth_LogoutClearsCookies(t *testing.T) {
	app := newTestApp(t)
	app.seedUser(t, "user-1", "jane@example.com")
	cookies := app.userCookies(t, "user-1", "jane@example.com")

	rec := app.do(t, http.MethodPost, "/api/auth/logout", nil, &cookies, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "tx_access_token" {
			found = true
			assert.True(t, c.MaxAge < 0, "logout must expire the access token cookie")
		}
	}
	assert.True(t, found, "logout response must clear the access token cookie")
}

func TestE2E_AdminAuth_LoginSuccessAndFailure(t *testing.T) {
	app := newTestApp(t)
	app.seedAdmin(t, "admin-1", "admin@example.com", "correct-password", "admin")

	var success struct {
		Success bool `json:"success"`
		Data    struct {
			Admin dto.AdminResponse `json:"admin"`
		} `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/admin/auth/login",
		map[string]string{"email": "admin@example.com", "password": "correct-password"}, nil, &success)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "admin@example.com", success.Data.Admin.Email)

	var accessCookieSet bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "tx_admin_access_token" && c.Value != "" {
			accessCookieSet = true
		}
	}
	assert.True(t, accessCookieSet, "successful login must set the admin access cookie")

	rec = app.do(t, http.MethodPost, "/api/admin/auth/login",
		map[string]string{"email": "admin@example.com", "password": "wrong-password"}, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestE2E_AdminAuth_ProtectedRoutesRequireAdminAuth(t *testing.T) {
	app := newTestApp(t)

	rec := app.do(t, http.MethodGet, "/api/admin/auth/me", nil, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestE2E_AdminAuth_AdminCRUDFlow(t *testing.T) {
	app := newTestApp(t)
	app.seedAdmin(t, "admin-1", "super@example.com", "superpassword1", "superadmin")
	cookies := app.adminCookies(t, "admin-1", "super@example.com", "superadmin")

	var created struct {
		Success bool              `json:"success"`
		Data    dto.AdminResponse `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/admin/admins", map[string]string{
		"email": "new-admin@example.com", "name": "New Admin", "password": "newpassword1", "role": "admin",
	}, &cookies, &created)
	require.Equal(t, http.StatusCreated, rec.Code)
	newAdminID := created.Data.ID
	require.NotEmpty(t, newAdminID)

	var updated struct {
		Success bool              `json:"success"`
		Data    dto.AdminResponse `json:"data"`
	}
	rec = app.do(t, http.MethodPut, "/api/admin/admins/"+newAdminID, map[string]string{
		"name": "Renamed Admin",
	}, &cookies, &updated)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Renamed Admin", updated.Data.Name)

	rec = app.do(t, http.MethodDelete, "/api/admin/admins/"+newAdminID, nil, &cookies, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var list struct {
		Success bool                `json:"success"`
		Data    []dto.AdminResponse `json:"data"`
	}
	rec = app.do(t, http.MethodGet, "/api/admin/admins", nil, &cookies, &list)
	require.Equal(t, http.StatusOK, rec.Code)
	for _, a := range list.Data {
		assert.NotEqual(t, newAdminID, a.ID, "deleted admin must not appear in the list")
	}
}
