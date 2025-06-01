package http

import (
	"fmt"
	"time"
)

type SameSite int

const (
	SameSiteDefaultMode SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

type Cookie struct {
	Name  string
	Value string

	Path    string    // optional
	Domain  string    // optional
	Expires time.Time // optional

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	MaxAge      int
	Secure      bool
	HttpOnly    bool
	SameSite    SameSite
	Partitioned bool
}

func (c *Cookie) String() string {
	// todo support more
	return fmt.Sprintf("%s=%s", c.Name, c.Value)
}

func (c *Cookie) Parse(data string) error {
	// todo
	return nil
}
