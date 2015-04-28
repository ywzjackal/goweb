package goweb

import (
	"net/http"
)

var (
	SessionIdTag = "sid"
)

type Session interface {
	Init(*http.Request, http.ResponseWriter)
	Id() string
	Get(string) string
	Set(string, string)
	Remove(string)
}

type session struct {
	Session
	id             string
	request        *http.Request
	responseWriter http.ResponseWriter
}

func (s *session) New() Session {
	return &session{}
}

func (s *session) Init(req *http.Request, res http.ResponseWriter) {
	s.request = req
	s.responseWriter = res
	// init session id
	cookie, err := req.Cookie(SessionIdTag)
	if err != nil && cookie != nil && len(cookie.Value) != 0 {
		s.id = cookie.Value
	} else {
		// generate new session id
		s.id = generateSessionIdByRequest(req)
		cookie = &http.Cookie{
			Name:   SessionIdTag,
			Value:  s.id,
			MaxAge: 0, // means session cookie
			Path:   "/",
		}
		http.SetCookie(res, cookie)
	}
}

func (s *session) Id() string {
	return s.id
}

func (s *session) Get(key string) string {
	cookie, err := s.request.Cookie(key)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *session) Set(key, value string) {
	cookie := &http.Cookie{
		Name:   key,
		Value:  value,
		MaxAge: 0, // means session cookie
		Path:   "/",
	}
	http.SetCookie(s.responseWriter, cookie)
}
