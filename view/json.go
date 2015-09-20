package view

import (
	"encoding/json"
	"github.com/ywzjackal/goweb"
	"net/http"
)

type ViewJson int

func (v *ViewJson) Render(resp http.ResponseWriter, model interface{}) goweb.WebError {
	var (
		b   []byte = nil
		err error = nil
	)
	b, err = json.Marshal(model)
	if err != nil {
		return goweb.NewWebError(http.StatusInternalServerError, err.Error())
	}
	//	h := resp.Header()
	//	h.Set("Cache-Control", "no-store, must-revalidate")
	//	h.Set("Pragma", "no-cache")
	//	h.Set("Content-Type", "application/json;charset=utf-8")
	//	resp.WriteHeader(http.StatusOK)
	_, err = resp.Write(b)
	return nil
}

func (v *ViewJson)RenderIndent(resp http.ResponseWriter, model interface{}) goweb.WebError {
	var (
		b   []byte = nil
		err error = nil
	)
	b, err = json.MarshalIndent(model, "", " ")
	if err != nil {
		return goweb.NewWebError(http.StatusInternalServerError, err.Error())
	}
	//	h := resp.Header()
	//	h.Set("Cache-Control", "no-store, must-revalidate")
	//	h.Set("Pragma", "no-cache")
	//	h.Set("Content-Type", "application/json;charset=utf-8")
	//	resp.WriteHeader(http.StatusOK)
	_, err = resp.Write(b)
	return nil
}