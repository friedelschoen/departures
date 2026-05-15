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
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

type SQLArgumentFunc func(req *http.Request, params map[string]string) ([]any, error)

type SQLHandler struct {
	Methods  []string
	Endpoint string
	Title    string
	Query    string
	Single   bool
	ArgsFn   SQLArgumentFunc

	schemaEndpoint string
	schema         jsonschema.Schema
}

func (s SQLHandler) prepareQuery(srv Server, req *http.Request, params map[string]string) (*sql.Rows, error) {
	var ar []any
	if s.ArgsFn != nil {
		var err error
		ar, err = s.ArgsFn(req, params)
		if err != nil {
			return nil, err
		}
	}
	return srv.db.Query(s.Query, ar...)
}

func (s SQLHandler) getSQLResult(rows *sql.Rows) (any, error) {
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	row := make([]any, len(names))
	out := []map[string]any{}
	for rows.Next() {
		for i := range row {
			row[i] = new(any)
		}
		if err := rows.Scan(row...); err != nil {
			log.Fatalln(err)
		}
		dest := make(map[string]any, len(names))
		for i, key := range names {
			v := row[i]
			switch x := v.(type) {
			case []byte:
				dest[key] = string(x)
			default:
				dest[key] = x
			}
		}
		out = append(out, dest)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if s.Single {
		if len(out) == 0 {
			return nil, StatusError(http.StatusNotFound)
		}
		return out[0], nil
	}
	return out, nil
}

func jsonType(t string) string {
	switch strings.ToUpper(t) {
	case "CHAR",
		"BPCHAR",
		"VARCHAR",
		"TEXT",
		"NAME",
		"UUID",
		"INET",
		"CIDR",
		"MACADDR",
		"MACADDR8",
		"XML",
		"JSON",
		"JSONB",
		"DATE",
		"TIME",
		"TIMETZ",
		"TIMESTAMP",
		"TIMESTAMPTZ",
		"INTERVAL":
		return "string"

	case "INT2",
		"SMALLINT",
		"INT4",
		"INT",
		"INTEGER",
		"SERIAL",
		"SERIAL4",
		"INT8",
		"BIGINT",
		"BIGSERIAL",
		"SERIAL8":
		return "integer"

	case "FLOAT4",
		"REAL",
		"FLOAT8",
		"DOUBLE PRECISION",
		"NUMERIC",
		"DECIMAL",
		"MONEY":
		return "number"

	case "BOOL",
		"BOOLEAN":
		return "boolean"

	case "BYTEA":
		return "string" /* base64 or escaped byte string */

	default:
		return "string" /* unknown / any */
	}
}

func (s *SQLHandler) writeSchema(w http.ResponseWriter, rows *sql.Rows) {
	if s.schema.Type == "" {
		s.schema.Type = "object"
		s.schema.Properties = make(map[string]*jsonschema.Schema)

		columns, err := rows.ColumnTypes()
		if err != nil {
			log.Fatalln(err)
		}
		for _, t := range columns {
			var st jsonschema.Schema

			if nullable, _ := t.Nullable(); nullable {
				st.Types = []string{
					jsonType(t.DatabaseTypeName()),
					"null",
				}
			} else {
				st.Type = jsonType(t.DatabaseTypeName())
			}
			if l, ok := t.Length(); ok && l != math.MaxInt64 {
				st.MaxLength = new(int(l))
			}
			if prec, scale, ok := t.DecimalSize(); ok {
				fmt.Printf("prec=%d, scale=%d\n", prec, scale)
			}

			s.schema.Properties[t.Name()] = &st
			s.schema.Required = append(s.schema.Required, t.Name())
		}

		s.schema.Schema = "https://json-schema.org/draft/2020-12/schema"
		s.schema.Comment = fmt.Sprintf("Source at %s", os.Getenv("SOURCE_REPO"))
		s.schema.Title = s.Title
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(s.schema); err != nil {
		log.Println("unable to encode JSON: ", err)
	}
}

func (s SQLHandler) Handle(srv Server, w http.ResponseWriter, req *http.Request) bool {
	if s.Title != "" && s.schemaEndpoint == "" {
		s.schemaEndpoint = fmt.Sprintf("/api/schema/%s.json", s.Title)
	}
	if s.schemaEndpoint != "" && strings.Trim(s.schemaEndpoint, "/") == strings.Trim(req.URL.Path, "/") {
		rows, err := s.prepareQuery(srv, req, nil)
		if err != nil {
			log.Println(err)
			writeResponse(w, nil, http.StatusInternalServerError)
			return true
		}
		defer rows.Close()
		s.writeSchema(w, rows)
		return true
	}

	params, ok := s.Match(req.Method, req.URL.Path)
	if !ok {
		return false
	}

	rows, err := s.prepareQuery(srv, req, params)
	if err != nil {
		log.Println(err)
		writeResponse(w, nil, http.StatusInternalServerError)
		return true
	}
	defer rows.Close()

	result, err := s.getSQLResult(rows)

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
	writeResponse(w, result, status)
	return true
}
