package auth

type Authenticator interface {
	ValidateAdminToken(token string) bool
	ValidateUserToken(token string) bool
}
