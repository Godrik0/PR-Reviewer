package auth

type StaticTokenAuth struct {
	adminToken string
	userToken  string
}

func NewStaticTokenAuth(adminToken, userToken string) *StaticTokenAuth {
	return &StaticTokenAuth{
		adminToken: adminToken,
		userToken:  userToken,
	}
}

func (a *StaticTokenAuth) ValidateAdminToken(token string) bool {
	return token != "" && token == a.adminToken
}

func (a *StaticTokenAuth) ValidateUserToken(token string) bool {
	return token != "" && (token == a.adminToken || token == a.userToken)
}
