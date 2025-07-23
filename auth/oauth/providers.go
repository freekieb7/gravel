package oauth

import "time"

// OAuthProvider represents a generic OAuth provider interface
type OAuthProvider interface {
	Token() (*TokenResponse, error)
	TokenString() (string, error)
	RefreshToken(refreshToken string) (*TokenResponse, error)
	GetOpenIDConfiguration() (*OpenIDConfiguration, error)
	GetJWKS() (*JWKSet, error)
	ValidateToken(tokenString string) (*JWTPayload, error)
	ValidateTokenWithoutSignature(tokenString string) (*JWTPayload, error)
	IsTokenValid(tokenString string) bool
	GetTokenClaims(tokenString string) (*JWTPayload, error)
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IdToken      string `json:"id_token,omitempty"`
	// Internal fields for better token management
	IssuedAt  time.Time `json:"-"`
	ExpiresAt time.Time `json:"-"`
}

// IsExpired checks if the token has expired
func (t *TokenResponse) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// TimeUntilExpiry returns the duration until token expires
func (t *TokenResponse) TimeUntilExpiry() time.Duration {
	if t.IsExpired() {
		return 0
	}
	return time.Until(t.ExpiresAt)
}

// OpenIDConfiguration represents the OpenID Connect discovery document
type OpenIDConfiguration struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	JwksURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

// JWKSet represents a JSON Web Key Set
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string   `json:"kty"`
	Use string   `json:"use"`
	Kid string   `json:"kid"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c,omitempty"`
	X5t string   `json:"x5t,omitempty"`
}

// JWTHeader represents the header of a JWT token
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

// JWTPayload represents the payload of a JWT token
type JWTPayload struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Aud   string `json:"aud"`
	Exp   int64  `json:"exp"`
	Nbf   int64  `json:"nbf,omitempty"`
	Iat   int64  `json:"iat,omitempty"`
	Jti   string `json:"jti,omitempty"`
	Scope string `json:"scope,omitempty"`
	// Google-specific claims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	// Auth0-specific claims
	Nickname string `json:"nickname,omitempty"`
}
