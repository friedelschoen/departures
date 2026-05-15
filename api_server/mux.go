/*
Copyright (C) 2026 Friedel Schön

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

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

type APIHandler interface {
	Handle(srv Server, w http.ResponseWriter, req *http.Request) bool
}

type Response struct {
	SourceRepo string `json:"source_repo"`
	Status     int    `json:"status"`
	Result     any    `json:"result,omitempty"`
}

func writeResponse(w http.ResponseWriter, result any, status int) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := Response{
		SourceRepo: os.Getenv("SOURCE_REPO"),
		Status:     status,
		Result:     result,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(response); err != nil {
		log.Println("unable to encode JSON: ", err)
	}
}

type APIHandleMux []APIHandler

type Handler struct {
	Server Server
	Mux    APIHandleMux
}

func (s SQLHandler) Match(method string, target string) (map[string]string, bool) {
	if len(s.Methods) > 0 && !slices.Contains(s.Methods, method) {
		return nil, false
	}

	targetS := strings.Split(strings.Trim(target, "/"), "/")
	matchS := strings.Split(strings.Trim(s.Endpoint, "/"), "/")
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

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range h.Mux {
		if ok := route.Handle(h.Server, w, req); ok {
			return
		}
	}

	writeResponse(w, nil, http.StatusNotFound)
}
