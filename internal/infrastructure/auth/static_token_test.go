package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStaticTokenAuth(t *testing.T) {
	adminToken := "admin-token"
	userToken := "user-token"

	auth := NewStaticTokenAuth(adminToken, userToken)

	assert.NotNil(t, auth)
	assert.Equal(t, adminToken, auth.adminToken)
	assert.Equal(t, userToken, auth.userToken)
}

func TestStaticTokenAuth_ValidateAdminToken(t *testing.T) {
	auth := NewStaticTokenAuth("admin-secret", "user-secret")

	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "valid admin token",
			token:    "admin-secret",
			expected: true,
		},
		{
			name:     "invalid token",
			token:    "wrong-token",
			expected: false,
		},
		{
			name:     "empty token",
			token:    "",
			expected: false,
		},
		{
			name:     "user token instead of admin",
			token:    "user-secret",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.ValidateAdminToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticTokenAuth_ValidateUserToken(t *testing.T) {
	auth := NewStaticTokenAuth("admin-secret", "user-secret")

	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "valid user token",
			token:    "user-secret",
			expected: true,
		},
		{
			name:     "valid admin token",
			token:    "admin-secret",
			expected: true,
		},
		{
			name:     "invalid token",
			token:    "wrong-token",
			expected: false,
		},
		{
			name:     "empty token",
			token:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.ValidateUserToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}
