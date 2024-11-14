package session

/*
Inspited by https://github.com/symfony/symfony/blob/7.2/src/Symfony/Component/HttpFoundation/Session/SessionInterface.php
*/
type Session interface {
	GetId() string
	GetName() string
	Has(name string) bool
	Get(name string, fallback any) any
	Set(name string, value any)
	All() map[string]any
	Replace(attributes map[string]any)
	Remove(name string)
	Clear()
}

type defaultSession struct {
	id         string
	name       string
	attributes map[string]any
}

func NewDefaultSession(id, name string, attributes map[string]any) Session {
	return &defaultSession{
		id:         id,
		name:       name,
		attributes: attributes,
	}
}

func (s *defaultSession) GetId() string {
	return s.id
}

func (s *defaultSession) GetName() string {
	return s.name
}

func (s *defaultSession) All() map[string]any {
	return s.attributes
}

func (s *defaultSession) Clear() {
	s.attributes = make(map[string]any)
}

func (s *defaultSession) Get(name string, fallback any) any {
	value, found := s.attributes[name]
	if !found {
		return fallback
	}

	return value
}

func (s *defaultSession) Has(name string) bool {
	_, found := s.attributes[name]
	return found
}

func (s *defaultSession) Remove(name string) {
	delete(s.attributes, name)
}

func (s *defaultSession) Replace(attributes map[string]any) {
	s.attributes = attributes
}

func (s *defaultSession) Set(name string, value any) {
	s.attributes[name] = value
}
