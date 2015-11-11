package session

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/ywzjackal/goweb"
)

const (
	SessionIdTag   = "__sid__"
	SessionTimeout = time.Minute * 30
)

type session struct {
	goweb.Session
	id  string
	req *http.Request
	res http.ResponseWriter
	mem map[interface{}]interface{}
}

func NewSession(res http.ResponseWriter, req *http.Request, storage goweb.Storage) goweb.Session {
	s := &session{
		res: res,
		req: req,
	}
	cookie, err := req.Cookie(SessionIdTag)
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
		mem := make(map[interface{}]interface{})
		storage.SetWithLife(s.id, mem, SessionTimeout)
		s.mem = mem
	} else {
		s.mem = itfs.(map[interface{}]interface{})
	}
	return s
}

func (s *session) Id() string {
	return s.id
}

func (s *session) MemMap() map[interface{}]interface{} {
	return s.mem
}

func (s *session) Get(key string) string {
	cookie, err := s.req.Cookie(key)
	if err != nil {
		goweb.Err.Printf("Get cookie failed!%s", err.Error())
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

func generateSessionIdByRequest(req *http.Request) string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}
