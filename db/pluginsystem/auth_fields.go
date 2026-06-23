package pluginsystem

const (
	AuthFieldAccessToken       = "accessToken"
	AuthFieldRefreshToken      = "refreshToken"
	AuthFieldClientSecret      = "clientSecret"
	AuthFieldOAuthState        = "oauthState"
	AuthFieldOAuthCodeVerifier = "oauthCodeVerifier"
	AuthFieldOAuthRedirectURI  = "oauthRedirectURI"
)

func InternalAuthSecretFields() []string {
	return []string{
		AuthFieldAccessToken,
		AuthFieldRefreshToken,
		AuthFieldClientSecret,
		AuthFieldOAuthState,
		AuthFieldOAuthCodeVerifier,
	}
}

func InternalOAuthTransientFields() []string {
	return []string{
		AuthFieldOAuthState,
		AuthFieldOAuthCodeVerifier,
		AuthFieldOAuthRedirectURI,
	}
}

func PluginInputAuthBlockedFields() []string {
	return []string{
		AuthFieldRefreshToken,
		AuthFieldClientSecret,
		AuthFieldOAuthState,
		AuthFieldOAuthCodeVerifier,
		AuthFieldOAuthRedirectURI,
	}
}
