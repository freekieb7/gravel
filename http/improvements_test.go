package http

import (
	"bufio"
	"strings"
	"testing"
	"time"
)

func TestCookieString(t *testing.T) {
	cookie := &Cookie{
		Name:     "test",
		Value:    "value",
		Path:     "/",
		Domain:   "example.com",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: SameSiteLaxMode,
	}

	expected := "test=value; Path=/; Domain=example.com; Max-Age=3600; Secure; HttpOnly; SameSite=Lax"
	result := cookie.String()

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestCookieParse(t *testing.T) {
	cookieStr := "test=value; Path=/; Domain=example.com; Max-Age=3600; Secure; HttpOnly; SameSite=Lax"

	cookie := &Cookie{}
	err := cookie.Parse(cookieStr)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if cookie.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", cookie.Name)
	}
	if cookie.Value != "value" {
		t.Errorf("Expected value 'value', got '%s'", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("Expected path '/', got '%s'", cookie.Path)
	}
	if cookie.MaxAge != 3600 {
		t.Errorf("Expected MaxAge 3600, got %d", cookie.MaxAge)
	}
	if !cookie.Secure {
		t.Error("Expected Secure to be true")
	}
	if !cookie.HttpOnly {
		t.Error("Expected HttpOnly to be true")
	}
	if cookie.SameSite != SameSiteLaxMode {
		t.Errorf("Expected SameSite Lax, got %d", cookie.SameSite)
	}
}

func TestCookieValid(t *testing.T) {
	// Valid cookie
	cookie := &Cookie{
		Name:  "valid",
		Value: "test",
	}
	if err := cookie.Valid(); err != nil {
		t.Errorf("Valid cookie should not return error: %v", err)
	}

	// Invalid - empty name
	cookie = &Cookie{
		Name:  "",
		Value: "test",
	}
	if err := cookie.Valid(); err == nil {
		t.Error("Empty name should return error")
	}

	// Invalid - SameSite=None without Secure
	cookie = &Cookie{
		Name:     "test",
		Value:    "value",
		SameSite: SameSiteNoneMode,
		Secure:   false,
	}
	if err := cookie.Valid(); err == nil {
		t.Error("SameSite=None without Secure should return error")
	}
}

func TestCookieIsExpired(t *testing.T) {
	// Not expired
	cookie := &Cookie{
		Name:    "test",
		Value:   "value",
		Expires: time.Now().Add(time.Hour),
	}
	if cookie.IsExpired() {
		t.Error("Cookie should not be expired")
	}

	// Expired by time
	cookie = &Cookie{
		Name:    "test",
		Value:   "value",
		Expires: time.Now().Add(-time.Hour),
	}
	if !cookie.IsExpired() {
		t.Error("Cookie should be expired")
	}

	// Expired by MaxAge
	cookie = &Cookie{
		Name:   "test",
		Value:  "value",
		MaxAge: -1,
	}
	if !cookie.IsExpired() {
		t.Error("Cookie with MaxAge=-1 should be expired")
	}
}

func TestResponseSetCookie(t *testing.T) {
	response := &Response{}
	response.Reset()

	cookie := &Cookie{
		Name:  "test",
		Value: "value",
		Path:  "/",
	}

	response.SetCookie(cookie)

	// Check if Set-Cookie header was added
	found := false
	for i := 0; i < response.headerCount; i++ {
		header := &response.headers[i]
		headerName := string(header.Name[:header.NameLen])
		if strings.ToLower(headerName) == "set-cookie" {
			found = true
			headerValue := string(header.Value[:header.ValueLen])
			if !strings.Contains(headerValue, "test=value") {
				t.Errorf("Cookie header should contain 'test=value', got: %s", headerValue)
			}
			break
		}
	}

	if !found {
		t.Error("Set-Cookie header not found")
	}
}

func TestRequestCookie(t *testing.T) {
	// Create a mock request with cookie header
	reqData := "GET / HTTP/1.1\r\nCookie: test=value; other=data\r\n\r\n"
	buf := bufio.NewReader(strings.NewReader(reqData))

	req := &Request{}
	req.Reset()
	err := req.Parse(buf)
	if err != nil {
		t.Errorf("Request parse failed: %v", err)
	}

	// Test getting existing cookie
	cookie, err := req.Cookie([]byte("test"))
	if err != nil {
		t.Errorf("Failed to get cookie: %v", err)
	}
	if cookie.Name != "test" {
		t.Errorf("Expected cookie name 'test', got '%s'", cookie.Name)
	}
	if cookie.Value != "value" {
		t.Errorf("Expected cookie value 'value', got '%s'", cookie.Value)
	}

	// Test getting non-existent cookie
	_, err = req.Cookie([]byte("nonexistent"))
	if err != ErrNoCookie {
		t.Errorf("Expected ErrNoCookie, got %v", err)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test GetMimeType
	mimeType := GetMimeType("test.html")
	if mimeType != "text/html; charset=utf-8" {
		t.Errorf("Expected HTML mime type, got %s", mimeType)
	}

	mimeType = GetMimeType("test.unknown")
	if mimeType != "application/octet-stream" {
		t.Errorf("Expected octet-stream for unknown extension, got %s", mimeType)
	}

	// Test ValidateMethod
	if !ValidateMethod([]byte("GET")) {
		t.Error("GET should be valid method")
	}
	if ValidateMethod([]byte("INVALID")) {
		t.Error("INVALID should not be valid method")
	}
}

func TestResponseWithMethods(t *testing.T) {
	response := &Response{}
	response.Reset()

	// Test WithHTML
	response.WithHTML("<h1>Hello</h1>")
	if string(response.Body) != "<h1>Hello</h1>" {
		t.Error("WithHTML should set body")
	}

	// Test WithRedirect
	response.Reset()
	response.WithRedirect("/login", StatusFound)
	if response.Status != StatusFound {
		t.Errorf("Expected status %d, got %d", StatusFound, response.Status)
	}
}
