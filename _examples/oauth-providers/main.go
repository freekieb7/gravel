package main

import (
	"fmt"
	"log"

	"github.com/freekieb7/gravel/auth/oauth"
)

func main() {
	// Example 1: Using Microsoft OAuth
	fmt.Println("=== Microsoft OAuth Example ===")
	microsoftClient := oauth.NewMicrosoftClient("your-client-id", "your-client-secret", "your-tenant-id")

	// Get token
	token, err := microsoftClient.Token()
	if err != nil {
		log.Printf("Error getting Microsoft token: %v", err)
	} else {
		fmt.Printf("Microsoft Access Token: %s...\n", token.AccessToken[:20])
	}

	// Example 2: Using Google OAuth
	fmt.Println("\n=== Google OAuth Example ===")
	googleClient := oauth.NewGoogleClient("your-google-client-id", "your-google-client-secret")

	token, err = googleClient.Token()
	if err != nil {
		log.Printf("Error getting Google token: %v", err)
	} else {
		fmt.Printf("Google Access Token: %s...\n", token.AccessToken[:20])
	}

	// Example 3: Using Auth0 OAuth
	fmt.Println("\n=== Auth0 OAuth Example ===")
	auth0Client := oauth.NewAuth0Client("your-auth0-client-id", "your-auth0-client-secret", "your-domain.auth0.com", "your-api-audience")

	token, err = auth0Client.Token()
	if err != nil {
		log.Printf("Error getting Auth0 token: %v", err)
	} else {
		fmt.Printf("Auth0 Access Token: %s...\n", token.AccessToken[:20])
	}

	// Example 4: Using the factory function
	fmt.Println("\n=== Factory Function Example ===")

	// Microsoft via factory
	microsoftProvider, err := oauth.NewOAuthProvider("microsoft", "client-id", "client-secret", map[string]string{
		"tenant_id": "your-tenant-id",
	})
	if err != nil {
		log.Printf("Error creating Microsoft provider: %v", err)
	} else {
		fmt.Printf("Created Microsoft provider: %T\n", microsoftProvider)
	}

	// Google via factory
	googleProvider, err := oauth.NewOAuthProvider("google", "client-id", "client-secret", nil)
	if err != nil {
		log.Printf("Error creating Google provider: %v", err)
	} else {
		fmt.Printf("Created Google provider: %T\n", googleProvider)
	}

	// Auth0 via factory
	auth0Provider, err := oauth.NewOAuthProvider("auth0", "client-id", "client-secret", map[string]string{
		"domain":   "your-domain.auth0.com",
		"audience": "your-api-audience",
	})
	if err != nil {
		log.Printf("Error creating Auth0 provider: %v", err)
	} else {
		fmt.Printf("Created Auth0 provider: %T\n", auth0Provider)
	}

	// Example 5: Token validation workflow
	fmt.Println("\n=== Token Validation Example ===")

	// Get OpenID configuration
	config, err := microsoftClient.GetOpenIDConfiguration()
	if err != nil {
		log.Printf("Error getting OpenID config: %v", err)
	} else {
		fmt.Printf("Issuer: %s\n", config.Issuer)
		fmt.Printf("Token Endpoint: %s\n", config.TokenEndpoint)
		fmt.Printf("JWKS URI: %s\n", config.JwksURI)
	}

	// Get JWKS
	jwks, err := microsoftClient.GetJWKS()
	if err != nil {
		log.Printf("Error getting JWKS: %v", err)
	} else {
		fmt.Printf("Found %d keys in JWKS\n", len(jwks.Keys))
		for i, key := range jwks.Keys {
			fmt.Printf("  Key %d: ID=%s, Type=%s, Use=%s\n", i+1, key.Kid, key.Kty, key.Use)
		}
	}

	// Example 6: JWT Token Validation with Signature Verification
	fmt.Println("\n=== JWT Signature Verification Example ===")

	// Note: In real usage, you would have a valid JWT token from the OAuth flow
	// This is just for demonstration of the validation methods available

	fmt.Println("Available validation methods:")
	fmt.Println("1. ValidateToken(token) - Full validation with signature verification")
	fmt.Println("2. ValidateTokenWithoutSignature(token) - Validation without signature check (dev/testing)")
	fmt.Println("3. IsTokenValid(token) - Quick boolean check")
	fmt.Println("4. GetTokenClaims(token) - Extract claims without validation")

	// Example with a sample JWT structure (not a real token)
	sampleJWT := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InNhbXBsZS1rZXkifQ.eyJpc3MiOiJodHRwczovL2V4YW1wbGUuY29tIiwic3ViIjoiMTIzNDU2Nzg5MCIsImF1ZCI6InNhbXBsZS1hdWRpZW5jZSIsImV4cCI6OTk5OTk5OTk5OSwiaWF0IjoxNjAwMDAwMDAwfQ.sample-signature"

	// Extract claims without validation (safe for demo)
	claims, err := microsoftClient.GetTokenClaims(sampleJWT)
	if err != nil {
		log.Printf("Error extracting claims: %v", err)
	} else {
		fmt.Printf("Sample token claims - Issuer: %s, Subject: %s, Audience: %s\n",
			claims.Iss, claims.Sub, claims.Aud)
	}

	fmt.Println("\n=== Security Features ===")
	fmt.Println("✅ RSA signature verification using JWKS")
	fmt.Println("✅ Token expiration checking (exp claim)")
	fmt.Println("✅ Not-before validation (nbf claim)")
	fmt.Println("✅ Key ID matching for correct public key selection")
	fmt.Println("✅ Thread-safe caching of JWKS and OpenID configuration")
	fmt.Println("✅ Comprehensive error handling with context")

	fmt.Println("\nMulti-provider OAuth system with signature verification ready!")
}
