package main

import (
	"fmt"
	"log"

	"github.com/freekieb7/gravel/auth/oauth"
)

func main() {
	// Demonstrate Microsoft OAuth client with all features
	demonstrateMicrosoftOAuth()

	// Demonstrate Google OAuth client
	demonstrateGoogleOAuth()

	// Demonstrate Auth0 client
	demonstrateAuth0OAuth()
}

func demonstrateMicrosoftOAuth() {
	fmt.Println("=== Microsoft OAuth Demo ===")

	// Create Microsoft OAuth client
	client := oauth.NewMicrosoftClient(
		"your-client-id",
		"your-client-secret",
		"your-tenant-id",
	)

	// Customize scopes
	client.SetScopes([]string{
		"https://graph.microsoft.com/.default",
		"openid",
		"profile",
		"email",
	})

	fmt.Printf("Configured scopes: %v\n", client.GetScopes())

	// Get access token
	tokenResponse, err := client.Token()
	if err != nil {
		log.Printf("Failed to get token: %v", err)
		return
	}

	fmt.Printf("Access token received (length: %d)\n", len(tokenResponse.AccessToken))
	fmt.Printf("Token type: %s\n", tokenResponse.TokenType)
	fmt.Printf("Expires in: %d seconds\n", tokenResponse.ExpiresIn)
	fmt.Printf("Token issued at: %v\n", tokenResponse.IssuedAt)
	fmt.Printf("Token expires at: %v\n", tokenResponse.ExpiresAt)

	// Check if token is expired
	if tokenResponse.IsExpired() {
		fmt.Println("Token is expired!")
	} else {
		fmt.Printf("Token valid for: %v\n", tokenResponse.TimeUntilExpiry())
	}

	// Validate token with signature verification
	claims, err := client.ValidateToken(tokenResponse.AccessToken)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
	} else {
		fmt.Println("Token signature validated successfully!")
		if claims.Sub != "" {
			fmt.Printf("Token subject: %s\n", claims.Sub)
		}
		if claims.Iss != "" {
			fmt.Printf("Token issuer: %s\n", claims.Iss)
		}
		if claims.Aud != "" {
			fmt.Printf("Token audience: %s\n", claims.Aud)
		}
	}

	// Demonstrate refresh token (if available)
	if tokenResponse.RefreshToken != "" {
		fmt.Println("Attempting token refresh...")
		newTokenResponse, err := client.RefreshToken(tokenResponse.RefreshToken)
		if err != nil {
			log.Printf("Token refresh failed: %v", err)
		} else {
			fmt.Printf("Token refreshed successfully! New token length: %d\n", len(newTokenResponse.AccessToken))
		}
	}

	fmt.Println()
}

func demonstrateGoogleOAuth() {
	fmt.Println("=== Google OAuth Demo ===")

	// Create Google OAuth client
	client := oauth.NewGoogleClient(
		"your-google-client-id",
		"your-google-client-secret",
	)

	// Set custom scopes for Google
	client.SetScopes([]string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
		"openid",
	})

	fmt.Printf("Google scopes: %v\n", client.GetScopes())

	// Get token (this would typically be from authorization code flow)
	tokenResponse, err := client.Token()
	if err != nil {
		log.Printf("Failed to get Google token: %v", err)
		return
	}

	fmt.Printf("Google token received, expires in: %d seconds\n", tokenResponse.ExpiresIn)

	// Token timing information
	if !tokenResponse.IsExpired() {
		fmt.Printf("Google token valid for: %v\n", tokenResponse.TimeUntilExpiry())
	}

	fmt.Println()
}

func demonstrateAuth0OAuth() {
	fmt.Println("=== Auth0 OAuth Demo ===")

	// Create Auth0 OAuth client
	client := oauth.NewAuth0Client(
		"your-auth0-client-id",
		"your-auth0-client-secret",
		"your-domain.auth0.com",
		"https://your-domain.auth0.com/api/v2/",
	)

	// Set Auth0 scopes
	client.SetScopes([]string{
		"openid",
		"profile",
		"email",
		"read:users",
	})

	fmt.Printf("Auth0 scopes: %v\n", client.GetScopes())

	// Get token
	tokenResponse, err := client.Token()
	if err != nil {
		log.Printf("Failed to get Auth0 token: %v", err)
		return
	}

	fmt.Printf("Auth0 token received, type: %s\n", tokenResponse.TokenType)

	// Demonstrate token validation
	isValid := client.IsTokenValid(tokenResponse.AccessToken)
	fmt.Printf("Auth0 token valid: %t\n", isValid)

	// Get token claims without signature verification (for demo)
	claims, err := client.GetTokenClaims(tokenResponse.AccessToken)
	if err != nil {
		log.Printf("Failed to get token claims: %v", err)
	} else {
		fmt.Printf("Token issuer: %s\n", claims.Iss)
		fmt.Printf("Token subject: %s\n", claims.Sub)
	}

	fmt.Println()
}
