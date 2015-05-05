package goweb

import (
	"net/http"
)

var (
	SessionIdTag = "sid"
)

type Session interface {
	Id() string
	Get(string) string
	Set(string, string)
	Remove(string)
}

type session struct {
	FactoryStateful
	Session
	Context
	id string
}

func (s *session) Init() {
	// init session id
	cookie, err := s.Request().Cookie(SessionIdTag)
	if err != nil && cookie != nil && len(cookie.Value) != 0 {
		s.id = cookie.Value
	} else {
		// generate new session id
		s.id = generateSessionIdByRequest(s.Request())
		cookie = &http.Cookie{
			Name:   SessionIdTag,
			Value:  s.id,
			MaxAge: 0, // means session cookie
			Path:   "/",
		}
		http.SetCookie(s.ResponseWriter(), cookie)
	}
}

func (s *session) Id() string {
	s.Init()
	return s.id
}

func (s *session) Get(key string) string {
	s.Init()
	cookie, err := s.Request().Cookie(key)
	if err != nil {
		Err.Printf("Get cookie failed!%s", err.Error())
		return ""
	}
	return cookie.Value
}

func (s *session) Set(key, value string) {
	s.Init()
	cookie := &http.Cookie{
		Name:   key,
		Value:  value,
		MaxAge: 0, // means session cookie
		Path:   "/",
	}
	http.SetCookie(s.ResponseWriter(), cookie)
}

func (s *session) Remove(key string) {
	s.Init()
	cookie := &http.Cookie{
		Name:   key,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	}
	http.SetCookie(s.ResponseWriter(), cookie)
}
