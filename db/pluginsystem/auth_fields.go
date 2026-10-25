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

// InternalOAuthMetadataFields are host-managed token metadata, rather than
// editable credentials. Omitted values survive settings saves; explicit nil
// values are consumed as removal instructions by the instance update hook.
func InternalOAuthMetadataFields() []string {
	return []string{
		AuthFieldExpiresAt,
		AuthFieldTokenType,
		AuthFieldOAuthContext,
		AuthFieldScope,
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
