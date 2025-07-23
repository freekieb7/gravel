package http

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SameSite int

const (
	SameSiteDefaultMode SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

var (
	ErrNoCookie      = errors.New("http: named cookie not present")
	ErrInvalidCookie = errors.New("http: invalid cookie format")
	ErrCookieTooLong = errors.New("http: cookie value too long")
)

type Cookie struct {
	Name  string
	Value string

	Path        string
	Domain      string
	Expires     time.Time
	MaxAge      int
	Secure      bool
	HttpOnly    bool
	SameSite    SameSite
	Partitioned bool
}

func (c *Cookie) String() string {
	var b strings.Builder

	// Name=Value (required)
	b.WriteString(c.Name)
	b.WriteByte('=')
	b.WriteString(c.Value)

	// Path
	if c.Path != "" {
		b.WriteString("; Path=")
		b.WriteString(c.Path)
	}

	// Domain
	if c.Domain != "" {
		b.WriteString("; Domain=")
		b.WriteString(c.Domain)
	}

	// Expires
	if !c.Expires.IsZero() {
		b.WriteString("; Expires=")
		b.WriteString(c.Expires.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	}

	// Max-Age
	if c.MaxAge > 0 {
		b.WriteString("; Max-Age=")
		b.WriteString(strconv.Itoa(c.MaxAge))
	} else if c.MaxAge < 0 {
		b.WriteString("; Max-Age=0")
	}

	// Secure
	if c.Secure {
		b.WriteString("; Secure")
	}

	// HttpOnly
	if c.HttpOnly {
		b.WriteString("; HttpOnly")
	}

	// SameSite
	switch c.SameSite {
	case SameSiteLaxMode:
		b.WriteString("; SameSite=Lax")
	case SameSiteStrictMode:
		b.WriteString("; SameSite=Strict")
	case SameSiteNoneMode:
		b.WriteString("; SameSite=None")
	}

	// Partitioned
	if c.Partitioned {
		b.WriteString("; Partitioned")
	}

	return b.String()
}

// Valid checks if the cookie is valid according to RFC 6265
func (c *Cookie) Valid() error {
	if c.Name == "" {
		return fmt.Errorf("cookie name cannot be empty")
	}

	// Check for invalid characters in name
	for _, r := range c.Name {
		if !isValidCookieNameChar(r) {
			return fmt.Errorf("invalid character in cookie name: %c", r)
		}
	}

	// Check value length (practical limit)
	if len(c.Value) > 4096 {
		return ErrCookieTooLong
	}

	// SameSite=None requires Secure
	if c.SameSite == SameSiteNoneMode && !c.Secure {
		return fmt.Errorf("SameSite=None requires Secure attribute")
	}

	return nil
}

// IsExpired returns true if the cookie has expired
func (c *Cookie) IsExpired() bool {
	if c.MaxAge < 0 {
		return true
	}
	if !c.Expires.IsZero() && c.Expires.Before(time.Now()) {
		return true
	}
	return false
}

// Clone creates a deep copy of the cookie
func (c *Cookie) Clone() *Cookie {
	return &Cookie{
		Name:        c.Name,
		Value:       c.Value,
		Path:        c.Path,
		Domain:      c.Domain,
		Expires:     c.Expires,
		MaxAge:      c.MaxAge,
		Secure:      c.Secure,
		HttpOnly:    c.HttpOnly,
		SameSite:    c.SameSite,
		Partitioned: c.Partitioned,
	}
}

// SetExpiry is a convenience method to set expiration
func (c *Cookie) SetExpiry(duration time.Duration) {
	c.Expires = time.Now().Add(duration)
	c.MaxAge = int(duration.Seconds())
}

// Delete marks the cookie for deletion
func (c *Cookie) Delete() {
	c.Value = ""
	c.MaxAge = -1
	c.Expires = time.Unix(1, 0) // January 1, 1970
}

func (c *Cookie) Parse(data string) error {
	// Reset cookie to defaults
	*c = Cookie{}

	// Split by semicolon and trim spaces
	parts := strings.Split(data, ";")
	if len(parts) == 0 {
		return fmt.Errorf("empty cookie string")
	}

	// First part is name=value
	nameValue := strings.TrimSpace(parts[0])
	eq := strings.IndexByte(nameValue, '=')
	if eq < 0 {
		return fmt.Errorf("missing '=' in cookie")
	}

	c.Name = strings.TrimSpace(nameValue[:eq])
	c.Value = strings.TrimSpace(nameValue[eq+1:])

	if c.Name == "" {
		return fmt.Errorf("empty cookie name")
	}

	// Parse attributes
	for i := 1; i < len(parts); i++ {
		attr := strings.TrimSpace(parts[i])
		if attr == "" {
			continue
		}

		// Check for key=value attributes
		if eq := strings.IndexByte(attr, '='); eq >= 0 {
			key := strings.ToLower(strings.TrimSpace(attr[:eq]))
			value := strings.TrimSpace(attr[eq+1:])

			switch key {
			case "path":
				c.Path = value
			case "domain":
				c.Domain = value
			case "expires":
				if expires, err := parseTime(value); err == nil {
					c.Expires = expires
				}
			case "max-age":
				if maxAge, err := strconv.Atoi(value); err == nil {
					c.MaxAge = maxAge
				}
			case "samesite":
				switch strings.ToLower(value) {
				case "lax":
					c.SameSite = SameSiteLaxMode
				case "strict":
					c.SameSite = SameSiteStrictMode
				case "none":
					c.SameSite = SameSiteNoneMode
				default:
					c.SameSite = SameSiteDefaultMode
				}
			}
		} else {
			// Boolean attributes
			switch strings.ToLower(attr) {
			case "secure":
				c.Secure = true
			case "httponly":
				c.HttpOnly = true
			case "partitioned":
				c.Partitioned = true
			}
		}
	}

	return nil
}

// parseTime attempts to parse cookie expiration time in various formats
func parseTime(value string) (time.Time, error) {
	// Common cookie time formats
	formats := []string{
		"Mon, 02 Jan 2006 15:04:05 GMT",
		"Mon, 02-Jan-2006 15:04:05 GMT",
		"Mon, 02-Jan-06 15:04:05 GMT",
		"Monday, 02-Jan-06 15:04:05 GMT",
		"Mon Jan 02 15:04:05 2006",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", value)
}

// isValidCookieNameChar returns true if the character is valid in a cookie name
func isValidCookieNameChar(r rune) bool {
	// RFC 6265 - valid characters for cookie names
	return r > 0x20 && r < 0x7f && r != '"' && r != ',' && r != ';' && r != '\\' &&
		r != '=' && r != '(' && r != ')' && r != '<' && r != '>' && r != '@' &&
		r != '{' && r != '}' && r != '[' && r != ']' && r != '?' && r != ':' && r != '/'
}

// ParseCookies parses multiple cookies from a Cookie header value
func ParseCookies(cookieHeader string) ([]*Cookie, error) {
	var cookies []*Cookie

	// Split by semicolon but be careful about quoted values
	parts := strings.Split(cookieHeader, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		cookie := &Cookie{}
		if err := cookie.Parse(part); err != nil {
			continue // Skip invalid cookies rather than failing entirely
		}

		cookies = append(cookies, cookie)
	}

	return cookies, nil
}
