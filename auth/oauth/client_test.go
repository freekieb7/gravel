package oauth

import (
	"testing"
	"time"
)

func TestNewMicrosoftClient(t *testing.T) {
	client := NewMicrosoftClient("test-client-id", "test-secret", "test-tenant")

	if client.ClientId != "test-client-id" {
		t.Errorf("Expected ClientId 'test-client-id', got '%s'", client.ClientId)
	}

	if client.ClientSecret != "test-secret" {
		t.Errorf("Expected ClientSecret 'test-secret', got '%s'", client.ClientSecret)
	}

	if client.TenantId != "test-tenant" {
		t.Errorf("Expected TenantId 'test-tenant', got '%s'", client.TenantId)
	}

	expectedTokenUrl := "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/token"
	if client.TokenUrl != expectedTokenUrl {
		t.Errorf("Expected TokenUrl '%s', got '%s'", expectedTokenUrl, client.TokenUrl)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.httpClient.Timeout)
	}
}

func TestGetTokenClaims(t *testing.T) {
	client := NewMicrosoftClient("test", "test", "test")

	// Sample JWT token (this is just for testing structure, not a real token)
	// Header: {"alg":"RS256","typ":"JWT","kid":"test-key"}
	// Payload: {"iss":"test-issuer","sub":"test-subject","aud":"test-audience","exp":9999999999,"iat":1234567890}
	testToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJpc3MiOiJ0ZXN0LWlzc3VlciIsInN1YiI6InRlc3Qtc3ViamVjdCIsImF1ZCI6InRlc3QtYXVkaWVuY2UiLCJleHAiOjk5OTk5OTk5OTksImlhdCI6MTIzNDU2Nzg5MH0.signature"

	payload, err := client.GetTokenClaims(testToken)
	if err != nil {
		t.Errorf("GetTokenClaims failed: %v", err)
	}

	if payload.Iss != "test-issuer" {
		t.Errorf("Expected issuer 'test-issuer', got '%s'", payload.Iss)
	}

	if payload.Sub != "test-subject" {
		t.Errorf("Expected subject 'test-subject', got '%s'", payload.Sub)
	}

	if payload.Aud != "test-audience" {
		t.Errorf("Expected audience 'test-audience', got '%s'", payload.Aud)
	}
}

func TestParseRSAPublicKey(t *testing.T) {
	client := NewMicrosoftClient("test", "test", "test")

	// Sample JWK with RSA key (test data)
	jwk := &JWK{
		Kty: "RSA",
		Kid: "test-key",
		N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
		E:   "AQAB",
	}

	publicKey, err := client.parseRSAPublicKey(jwk)
	if err != nil {
		t.Errorf("parseRSAPublicKey failed: %v", err)
	}

	if publicKey == nil {
		t.Error("Expected public key, got nil")
	}

	if publicKey.E != 65537 { // AQAB in base64 is 65537
		t.Errorf("Expected exponent 65537, got %d", publicKey.E)
	}
}

func TestParseRSAPublicKey_InvalidKeyType(t *testing.T) {
	client := NewMicrosoftClient("test", "test", "test")

	jwk := &JWK{
		Kty: "EC", // Not RSA
		Kid: "test-key",
	}

	_, err := client.parseRSAPublicKey(jwk)
	if err == nil {
		t.Error("Expected error for non-RSA key type, got nil")
	}

	expectedError := "key type is not RSA"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateTokenWithoutSignature(t *testing.T) {
	client := NewMicrosoftClient("test", "test", "test")

	// Sample JWT token (this is just for testing structure, not a real token)
	// Header: {"alg":"RS256","typ":"JWT","kid":"test-key"}
	// Payload: {"iss":"test-issuer","sub":"test-subject","aud":"test-audience","exp":9999999999,"iat":1234567890}
	testToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJpc3MiOiJ0ZXN0LWlzc3VlciIsInN1YiI6InRlc3Qtc3ViamVjdCIsImF1ZCI6InRlc3QtYXVkaWVuY2UiLCJleHAiOjk5OTk5OTk5OTksImlhdCI6MTIzNDU2Nzg5MH0.signature"

	payload, err := client.ValidateTokenWithoutSignature(testToken)
	if err != nil {
		t.Errorf("ValidateTokenWithoutSignature failed: %v", err)
	}

	if payload.Iss != "test-issuer" {
		t.Errorf("Expected issuer 'test-issuer', got '%s'", payload.Iss)
	}

	if payload.Sub != "test-subject" {
		t.Errorf("Expected subject 'test-subject', got '%s'", payload.Sub)
	}
}

func TestIsTokenValid_InvalidFormat(t *testing.T) {
	client := NewMicrosoftClient("test", "test", "test")

	// Test invalid token format
	if client.IsTokenValid("invalid.token") {
		t.Error("Expected invalid token to return false")
	}

	if client.IsTokenValid("") {
		t.Error("Expected empty token to return false")
	}
}
