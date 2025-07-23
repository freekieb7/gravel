package oauth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BaseOAuthClient provides common OAuth functionality
type BaseOAuthClient struct {
	ClientId     string
	ClientSecret string
	TokenUrl     string
	ConfigUrl    string
	Scopes       []string
	httpClient   *http.Client
	jwksCache    *JWKSet
	configCache  *OpenIDConfiguration
	cacheMutex   sync.RWMutex
	cacheExpiry  time.Time
}

// NewBaseOAuthClient creates a new base OAuth client
func NewBaseOAuthClient(clientId, clientSecret, tokenUrl, configUrl string) *BaseOAuthClient {
	return &BaseOAuthClient{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		TokenUrl:     tokenUrl,
		ConfigUrl:    configUrl,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetScopes sets custom scopes for the OAuth client
func (client *BaseOAuthClient) SetScopes(scopes []string) {
	client.Scopes = scopes
}

// GetScopes returns the configured scopes
func (client *BaseOAuthClient) GetScopes() []string {
	return client.Scopes
}

// MicrosoftClient represents a Microsoft OAuth client
type MicrosoftClient struct {
	*BaseOAuthClient
	TenantId string
}

// GoogleClient represents a Google OAuth client
type GoogleClient struct {
	*BaseOAuthClient
}

// Auth0Client represents an Auth0 OAuth client
type Auth0Client struct {
	*BaseOAuthClient
	Domain   string
	Audience string
}

// NewMicrosoftClient creates a new Microsoft OAuth client
func NewMicrosoftClient(clientId, clientSecret, tenantId string) *MicrosoftClient {
	tokenUrl := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantId)
	configUrl := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/.well-known/openid_configuration", tenantId)

	return &MicrosoftClient{
		BaseOAuthClient: NewBaseOAuthClient(clientId, clientSecret, tokenUrl, configUrl),
		TenantId:        tenantId,
	}
}

// NewGoogleClient creates a new Google OAuth client
func NewGoogleClient(clientId, clientSecret string) *GoogleClient {
	tokenUrl := "https://oauth2.googleapis.com/token"
	configUrl := "https://accounts.google.com/.well-known/openid_configuration"

	return &GoogleClient{
		BaseOAuthClient: NewBaseOAuthClient(clientId, clientSecret, tokenUrl, configUrl),
	}
}

// NewAuth0Client creates a new Auth0 OAuth client
func NewAuth0Client(clientId, clientSecret, domain, audience string) *Auth0Client {
	tokenUrl := fmt.Sprintf("https://%s/oauth/token", domain)
	configUrl := fmt.Sprintf("https://%s/.well-known/openid_configuration", domain)

	return &Auth0Client{
		BaseOAuthClient: NewBaseOAuthClient(clientId, clientSecret, tokenUrl, configUrl),
		Domain:          domain,
		Audience:        audience,
	}
}

// GetOpenIDConfiguration fetches the OpenID Connect configuration
func (client *BaseOAuthClient) GetOpenIDConfiguration() (*OpenIDConfiguration, error) {
	client.cacheMutex.RLock()
	if client.configCache != nil && time.Now().Before(client.cacheExpiry) {
		defer client.cacheMutex.RUnlock()
		return client.configCache, nil
	}
	client.cacheMutex.RUnlock()

	resp, err := client.httpClient.Get(client.ConfigUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenID configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenID configuration request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenID configuration response: %w", err)
	}

	var config OpenIDConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse OpenID configuration: %w", err)
	}

	// Cache the configuration for 1 hour
	client.cacheMutex.Lock()
	client.configCache = &config
	client.cacheExpiry = time.Now().Add(time.Hour)
	client.cacheMutex.Unlock()

	return &config, nil
}

// GetJWKS fetches the JSON Web Key Set for token validation
func (client *BaseOAuthClient) GetJWKS() (*JWKSet, error) {
	config, err := client.GetOpenIDConfiguration()
	if err != nil {
		return nil, err
	}

	client.cacheMutex.RLock()
	if client.jwksCache != nil && time.Now().Before(client.cacheExpiry) {
		defer client.cacheMutex.RUnlock()
		return client.jwksCache, nil
	}
	client.cacheMutex.RUnlock()

	resp, err := client.httpClient.Get(config.JwksURI)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks JWKSet
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Cache the JWKS for 1 hour
	client.cacheMutex.Lock()
	client.jwksCache = &jwks
	client.cacheMutex.Unlock()

	return &jwks, nil
}

// parseRSAPublicKey converts a JWK to an RSA public key
func (client *BaseOAuthClient) parseRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, errors.New("key type is not RSA")
	}

	// Decode the modulus (n) and exponent (e) from base64url
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return publicKey, nil
}

// verifyJWTSignature verifies the JWT signature using RSA-SHA256
func (client *BaseOAuthClient) verifyJWTSignature(tokenParts []string, publicKey *rsa.PublicKey) error {
	if len(tokenParts) != 3 {
		return errors.New("invalid JWT format")
	}

	// Create the signing input (header.payload)
	signingInput := tokenParts[0] + "." + tokenParts[1]

	// Decode the signature
	signature, err := base64.RawURLEncoding.DecodeString(tokenParts[2])
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the signing input with SHA-256
	hash := sha256.Sum256([]byte(signingInput))

	// Verify the signature
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// ValidateTokenWithoutSignature validates a JWT token without signature verification (useful for development/testing)
func (client *BaseOAuthClient) ValidateTokenWithoutSignature(tokenString string) (*JWTPayload, error) {
	// Parse JWT token
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	// Check expiration
	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return nil, errors.New("token has expired")
	}

	// Check not before
	if payload.Nbf > 0 && time.Now().Unix() < payload.Nbf {
		return nil, errors.New("token not yet valid")
	}

	return &payload, nil
}

// ValidateToken validates a JWT token using the JWKS
func (client *BaseOAuthClient) ValidateToken(tokenString string) (*JWTPayload, error) {
	// Parse JWT token
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT header: %w", err)
	}

	var header JWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to parse JWT header: %w", err)
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	// Check expiration
	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return nil, errors.New("token has expired")
	}

	// Check not before
	if payload.Nbf > 0 && time.Now().Unix() < payload.Nbf {
		return nil, errors.New("token not yet valid")
	}

	// Get JWKS for signature validation
	jwks, err := client.GetJWKS()
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	// Find matching key
	var matchingKey *JWK
	for _, key := range jwks.Keys {
		if key.Kid == header.Kid {
			matchingKey = &key
			break
		}
	}

	if matchingKey == nil {
		return nil, errors.New("no matching key found for token")
	}

	// Parse the RSA public key
	publicKey, err := client.parseRSAPublicKey(matchingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
	}

	// Verify the JWT signature
	if err := client.verifyJWTSignature(parts, publicKey); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &payload, nil
}

// IsTokenValid is a convenience method to check if a token is valid
func (client *BaseOAuthClient) IsTokenValid(tokenString string) bool {
	_, err := client.ValidateToken(tokenString)
	return err == nil
}

// GetTokenClaims extracts claims from a token without full validation
func (client *BaseOAuthClient) GetTokenClaims(tokenString string) (*JWTPayload, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	return &payload, nil
}

// Microsoft Token implementation
func (client *MicrosoftClient) Token() (*TokenResponse, error) {
	response, err := client.httpClient.PostForm(client.TokenUrl, url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
	})
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				log.Printf("closing body error: %v", closeErr)
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, errors.New("access token not found in response")
	}

	return &tokenResponse, nil
}

// TokenString is a convenience method that returns just the access token string
func (client *MicrosoftClient) TokenString() (string, error) {
	tokenResponse, err := client.Token()
	if err != nil {
		return "", err
	}
	return tokenResponse.AccessToken, nil
}

// RefreshToken refreshes an access token using a refresh token
func (client *MicrosoftClient) RefreshToken(refreshToken string) (*TokenResponse, error) {
	response, err := client.httpClient.PostForm(client.TokenUrl, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
		"refresh_token": {refreshToken},
	})
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				log.Printf("closing body error: %v", closeErr)
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response body: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, errors.New("access token not found in refresh response")
	}

	// Set token timing information
	tokenResponse.IssuedAt = time.Now()
	if tokenResponse.ExpiresIn > 0 {
		tokenResponse.ExpiresAt = tokenResponse.IssuedAt.Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	}

	return &tokenResponse, nil
}

// Google Token implementation
func (client *GoogleClient) Token() (*TokenResponse, error) {
	response, err := client.httpClient.PostForm(client.TokenUrl, url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
		"scope":         {"https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"},
	})
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				log.Printf("closing body error: %v", closeErr)
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, errors.New("access token not found in response")
	}

	return &tokenResponse, nil
}

// TokenString is a convenience method that returns just the access token string
func (client *GoogleClient) TokenString() (string, error) {
	tokenResponse, err := client.Token()
	if err != nil {
		return "", err
	}
	return tokenResponse.AccessToken, nil
}

// RefreshToken refreshes an access token using a refresh token
func (client *GoogleClient) RefreshToken(refreshToken string) (*TokenResponse, error) {
	response, err := client.httpClient.PostForm(client.TokenUrl, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
		"refresh_token": {refreshToken},
	})
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				log.Printf("closing body error: %v", closeErr)
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response body: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, errors.New("access token not found in refresh response")
	}

	// Set token timing information
	tokenResponse.IssuedAt = time.Now()
	if tokenResponse.ExpiresIn > 0 {
		tokenResponse.ExpiresAt = tokenResponse.IssuedAt.Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	}

	return &tokenResponse, nil
}

// Auth0 Token implementation
func (client *Auth0Client) Token() (*TokenResponse, error) {
	values := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
	}

	// Add audience if specified
	if client.Audience != "" {
		values.Set("audience", client.Audience)
	}

	response, err := client.httpClient.PostForm(client.TokenUrl, values)
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				log.Printf("closing body error: %v", closeErr)
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, errors.New("access token not found in response")
	}

	return &tokenResponse, nil
}

// TokenString is a convenience method that returns just the access token string
func (client *Auth0Client) TokenString() (string, error) {
	tokenResponse, err := client.Token()
	if err != nil {
		return "", err
	}
	return tokenResponse.AccessToken, nil
}

// RefreshToken refreshes an access token using a refresh token
func (client *Auth0Client) RefreshToken(refreshToken string) (*TokenResponse, error) {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {client.ClientId},
		"client_secret": {client.ClientSecret},
		"refresh_token": {refreshToken},
	}

	response, err := client.httpClient.PostForm(client.TokenUrl, values)
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				log.Printf("closing body error: %v", closeErr)
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response body: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, errors.New("access token not found in refresh response")
	}

	// Set token timing information
	tokenResponse.IssuedAt = time.Now()
	if tokenResponse.ExpiresIn > 0 {
		tokenResponse.ExpiresAt = tokenResponse.IssuedAt.Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	}

	return &tokenResponse, nil
}

// Factory function for creating OAuth providers
func NewOAuthProvider(provider, clientId, clientSecret string, options map[string]string) (OAuthProvider, error) {
	switch strings.ToLower(provider) {
	case "microsoft":
		tenantId, ok := options["tenant_id"]
		if !ok {
			return nil, errors.New("tenant_id is required for Microsoft OAuth")
		}
		return NewMicrosoftClient(clientId, clientSecret, tenantId), nil

	case "google":
		return NewGoogleClient(clientId, clientSecret), nil

	case "auth0":
		domain, ok := options["domain"]
		if !ok {
			return nil, errors.New("domain is required for Auth0 OAuth")
		}
		audience := options["audience"] // Optional
		return NewAuth0Client(clientId, clientSecret, domain, audience), nil

	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}
