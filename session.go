package goweb

import (
	"net/http"
	"time"
)

var (
	SessionIdTag   = "__sid__"
	SessionTimeout = time.Minute * 1
)

type Session interface {
	Init(http.ResponseWriter, *http.Request, Storage) WebError
	Id() string
	Get(string) string
	Set(string, string)
	Remove(string)
	MemMap() map[string]interface{}
}

type session struct {
	Session
	id  string
	req *http.Request
	res http.ResponseWriter
	mem map[string]interface{}
}

func (s *session) Init(res http.ResponseWriter, req *http.Request, storage Storage) WebError {
	s.res = res
	s.req = req
	cookie, err := s.req.Cookie(SessionIdTag)
	if err == nil && cookie != nil && len(cookie.Value) != 0 {
		s.id = cookie.Value
	} else {
		// generate new session id
		s.id = generateSessionIdByRequest(s.req)
		cookie = &http.Cookie{
			Name:  SessionIdTag,
			Value: s.id,
		}
		http.SetCookie(s.res, cookie)
	}
	itfs := storage.Get(s.id)
	if itfs == nil {
		mem := make(map[string]interface{})
		storage.SetWithLife(s.id, mem, SessionTimeout)
		s.mem = mem
	} else {
		s.mem = itfs.(map[string]interface{})
	}
	return nil
}

func (s *session) Id() string {
	return s.id
}

func (s *session) MemMap() map[string]interface{} {
	return s.mem
}

func (s *session) Get(key string) string {
	cookie, err := s.req.Cookie(key)
	if err != nil {
		Err.Printf("Get cookie failed!%s", err.Error())
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
	http.SetCookie(s.res, cookie)
}

func (s *session) Remove(key string) {
	cookie := &http.Cookie{
		Name:   key,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	}
	http.SetCookie(s.res, cookie)
}
