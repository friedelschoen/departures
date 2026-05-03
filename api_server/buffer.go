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

	"github.com/dylandreimerink/go-rijksdriehoek"
)

//go:embed sql/buffer.sql
var bufferSQL string

func (s *Server) stationBuffer(r *http.Request, params map[string]string) (any, error) {
	rows, err := s.db.Query(bufferSQL, params["stop"])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Departure

	for rows.Next() {
		var (
			d           Departure
			punctuality int
			rdx, rdy    *int
		)

		if err := rows.Scan(
			&d.RouteID,
			&d.Line,
			&d.BGColor,
			&d.FGColor,
			&d.TripID,
			&d.Headsign,
			&d.Platform,
			&d.ScheduledTime,
			&d.Status,
			&d.LastSeen,
			&punctuality,
			&d.VehicleNumber,
			&d.BlockCode,
			&rdx,
			&rdy,
		); err != nil {
			return nil, err
		}

		d.DelayMinutes = -punctuality / 60
		d.RealtimeTime = d.ScheduledTime

		if rdx != nil && rdy != nil {
			d.Lat, d.Lon = rijksdriehoek.RDtoWGS84(float64(*rdx), float64(*rdy))
		}

		out = append(out, d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
