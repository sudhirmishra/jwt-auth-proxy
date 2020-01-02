package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestSignup(t *testing.T) {
	clearTestDB()

	// Send Signup request and check response
	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusCreated, res.Code)

	// Check mail content
	checkTestString(t, "foo@bar.com", smtpMockContent.RcptValue)
	if smtpMockContent.Buffer.DataValue == "" {
		t.Error("Expected ConfirmID as mail content")
	}

	// Check that login is not yet possible
	payload = `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)

	// Call confirm to activate account
	req, _ = http.NewRequest("POST", "/auth/confirm/"+smtpMockContent.Buffer.DataValue, nil)
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)

	// Check again that login is possible after confirmation
	payload = `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusOK, res.Code)
}

func TestAuthChangeEmail(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	// Init email change
	payload := "{\"email\": \"foo2@bar.com\", \"password\": \"12345678\"}"
	req := newHTTPRequest("POST", "/auth/changeemail", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)

	// Check mail content
	checkTestString(t, "foo2@bar.com", smtpMockContent.RcptValue)
	if smtpMockContent.Buffer.DataValue == "" {
		t.Error("Expected ConfirmID as mail content")
	}

	// Check that email is still old
	if GetUserRepository().GetByEmail("foo2@bar.com") != nil {
		t.Error("Expected user to still have old address")
	}

	// Call confirm to perform change
	req, _ = http.NewRequest("POST", "/auth/confirm/"+smtpMockContent.Buffer.DataValue, nil)
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)

	// Check that email is new one now
	if GetUserRepository().GetByEmail("foo2@bar.com") == nil {
		t.Error("Expected user to have new address")
	}
}

func TestForgotPassword(t *testing.T) {
	clearTestDB()
	user := createTestUser(true)

	// Init password reset
	payload := "{\"email\": \"" + user.Email + "\"}"
	req := newHTTPRequest("POST", "/auth/initpwreset", "", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)

	// Check mail content
	checkTestString(t, user.Email, smtpMockContent.RcptValue)
	if smtpMockContent.Buffer.DataValue == "" {
		t.Error("Expected ConfirmID as mail content")
	}

	// Call confirm to perform change
	req, _ = http.NewRequest("POST", "/auth/confirm/"+smtpMockContent.Buffer.DataValue, nil)
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)

	// Check mail content
	checkTestString(t, user.Email, smtpMockContent.RcptValue)
	if smtpMockContent.Buffer.DataValue == "" {
		t.Error("Expected new password as mail content")
	}
	if len(smtpMockContent.Buffer.DataValue) != 8 {
		t.Error("Expected new password to have length of 8")
	}

	// Check that login is possible with new password
	payload = "{\"email\": \"" + user.Email + "\", \"password\": \"" + smtpMockContent.Buffer.DataValue + "\"}"
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusOK, res.Code)
}

func TestLogin(t *testing.T) {
	clearTestDB()
	createTestUser(true)

	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusOK, res.Code)
}

func TestLoginWrongPassword(t *testing.T) {
	clearTestDB()
	createTestUser(true)

	payload := `{"email": "foo@bar.com", "password": "11111111"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestLoginEmptyPassword(t *testing.T) {
	clearTestDB()
	createTestUser(true)

	payload := `{"email": "foo@bar.com", "password": ""}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestLoginUnconfirmedAccount(t *testing.T) {
	clearTestDB()
	createTestUser(false)

	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestLoginDisabledAccount(t *testing.T) {
	clearTestDB()
	user := &User{
		Email:          "foo@bar.com",
		CreateDate:     time.Now(),
		HashedPassword: GetUserRepository().GetHashedPassword("12345678"),
		Confirmed:      true,
		Enabled:        false,
	}
	GetUserRepository().Create(user)

	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestLoginUnsetConfirmed(t *testing.T) {
	clearTestDB()
	user := &User{
		Email:          "foo@bar.com",
		CreateDate:     time.Now(),
		HashedPassword: GetUserRepository().GetHashedPassword("12345678"),
		Enabled:        true,
	}
	GetUserRepository().Create(user)

	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestLoginUnsetEnabled(t *testing.T) {
	clearTestDB()
	user := &User{
		Email:          "foo@bar.com",
		CreateDate:     time.Now(),
		HashedPassword: GetUserRepository().GetHashedPassword("12345678"),
		Confirmed:      true,
	}
	GetUserRepository().Create(user)

	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestLoginUnsetPassword(t *testing.T) {
	clearTestDB()
	user := &User{
		Email:      "foo@bar.com",
		CreateDate: time.Now(),
		Confirmed:  true,
		Enabled:    true,
	}
	GetUserRepository().Create(user)

	payload := `{"email": "foo@bar.com", "password": "12345678"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestLogout(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	// Log out
	payload := "{\"refreshToken\": \"" + loginResponse.RefreshToken + "\"}"
	req := newHTTPRequest("POST", "/auth/logout", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusNoContent, res.Code)
}

func TestRenew(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := "{\"refreshToken\": \"" + loginResponse.RefreshToken + "\"}"
	req := newHTTPRequest("POST", "/auth/renew", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusOK, res.Code)

	var loginResponse2 LoginResponse
	json.Unmarshal(res.Body.Bytes(), &loginResponse)
	if loginResponse.AccessToken == loginResponse2.AccessToken {
		t.Error("Got no new access token")
	}
}

func TestRenewWithInvalidRefreshToken(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := "{\"refreshToken\": \"xxxxxxxxxxxxxx\"}"
	req := newHTTPRequest("POST", "/auth/renew", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestRenewWithoutRefreshToken(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := "{}"
	req := newHTTPRequest("POST", "/auth/renew", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestPing(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	req := newHTTPRequest("GET", "/auth/ping", loginResponse.AccessToken, nil)
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)
}

func TestWithInvalidAuthHeader(t *testing.T) {
	clearTestDB()
	createLoginTestUser()

	req := newHTTPRequest("GET", "/auth/ping", "xxxxxxx", nil)
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestWithoutAuthHeader(t *testing.T) {
	clearTestDB()
	createLoginTestUser()

	req := newHTTPRequest("GET", "/auth/ping", "", nil)
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestChangePassword(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := `{"oldPassword": "12345678", "newPassword": "00000000"}`
	req := newHTTPRequest("POST", "/auth/setpw", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusNoContent, res.Code)

	payload = `{"email": "foo@bar.com", "password": "00000000"}`
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBufferString(payload))
	res = executePublicTestRequest(req)
	checkTestResponseCode(t, http.StatusOK, res.Code)
}

func TestChangePasswordInvalidOld(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := `{"oldPassword": "xxxxxxxx", "newPassword": "00000000"}`
	req := newHTTPRequest("POST", "/auth/setpw", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestChangePasswordInvalidAccessToken(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := `{"oldPassword": "xxxxxxxx", "newPassword": "00000000"}`
	req := newHTTPRequest("POST", "/auth/setpw", loginResponse.AccessToken+"x", bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestChangePasswordInvalidNewPassword(t *testing.T) {
	clearTestDB()
	loginResponse := createLoginTestUser()

	payload := `{"oldPassword": "xxxxxxxx", "newPassword": "xx"}`
	req := newHTTPRequest("POST", "/auth/setpw", loginResponse.AccessToken, bytes.NewBufferString(payload))
	res := executePublicTestRequest(req)

	checkTestResponseCode(t, http.StatusBadRequest, res.Code)
}