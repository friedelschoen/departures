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
	_ "embed"
	"net/http"
)

type Stop struct {
	ID       string  `json:"id"`
	Code     *string `json:"code"`
	Name     string  `json:"name"`
	Parent   *string `json:"parent"`
	Platform *string `json:"platform"`
}

//go:embed sql/stop_query.sql
var stopQuerySQL string

func (s *Server) stopQuery(r *http.Request, _ map[string]string) (any, error) {
	q := r.URL.Query().Get("q")
	if q == "" {
		return []Stop{}, nil
	}

	rows, err := s.db.Query(stopQuerySQL, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Stop

	for rows.Next() {
		var st Stop
		if err := rows.Scan(
			&st.ID,
			&st.Code,
			&st.Name,
			&st.Parent,
			&st.Platform,
		); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
