package storage

import "github.com/freekieb7/gravel/session"

type SessionStore interface {
	Close() error
	Has(id string) bool
	Get(id string) (map[string]any, error)
	Save(session session.Session) error
	Delete(id string) error
}
