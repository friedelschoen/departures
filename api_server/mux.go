package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
)

type StatusError int

func (err StatusError) Error() string {
	return http.StatusText(int(err))
}

type APIHandler struct {
	Methods []string
	URL     string
	Handler func(req *http.Request, placeholders map[string]string) (any, error)
}

func (h APIHandler) Match(method string, target string) (map[string]string, bool) {
	if len(h.Methods) > 0 && !slices.Contains(h.Methods, method) {
		return nil, false
	}
	targetS := strings.Split(strings.Trim(target, "/"), "/")
	matchS := strings.Split(strings.Trim(h.URL, "/"), "/")

	if len(targetS) != len(matchS) {
		return nil, false
	}

	placeholders := make(map[string]string)
	for i, m := range matchS {
		if len(m) == 0 {
			return nil, false
		}
		if m[0] == ':' {
			placeholders[m[1:]] = targetS[i]
		} else if m != targetS[i] {
			return nil, false
		}
	}
	return placeholders, true
}

type Response struct {
	SourceRepo string `json:"source_repo"`
	Status     int    `json:"status"`
	Result     any    `json:"result,omitempty"`
}

type APIHandleMux []APIHandler

func writeResponse(w http.ResponseWriter, result any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var response Response
	response.SourceRepo = os.Getenv("SOURCE_REPO")
	response.Status = status
	response.Result = result
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(response); err != nil {
		log.Println("unable to encode JSON: ", err)
	}
}

func (mux APIHandleMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, h := range mux {
		if pl, ok := h.Match(req.Method, req.URL.Path); ok {
			res, err := h.Handler(req, pl)

			var status int
			switch err := err.(type) {
			case StatusError:
				status = int(err)
			case nil:
				status = http.StatusOK
			default:
				status = http.StatusInternalServerError
				log.Printf("error while handling %s %v: %v\n", req.Method, req.URL, err)
			}
			writeResponse(w, res, status)
			return
		}
	}
	writeResponse(w, nil, http.StatusNotFound)
}
