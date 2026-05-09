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
	_ "embed"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"strconv"

	"github.com/dylandreimerink/go-rijksdriehoek"
	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
)

//go:embed sql/vehicle_trip_shape.sql
var vehicleTripShapeSQL string

type MapPoint struct {
	Lat float64
	Lon float64
	Seq int
}

func writeMapPNG(w http.ResponseWriter, zoom int, markers *s2.LatLng, shape []MapPoint) error {
	ctx := sm.NewContext()
	ctx.SetSize(1200, 600)
	if zoom > 0 {
		ctx.SetZoom(zoom)
	}

	if len(shape) >= 2 {
		var pts []s2.LatLng
		for _, p := range shape {
			pts = append(pts, s2.LatLngFromDegrees(p.Lat, p.Lon))
		}
		ctx.AddObject(sm.NewPath(pts, color.RGBA{0, 80, 220, 255}, 3))
	}

	if markers != nil {
		ctx.SetCenter(*markers)
		ctx.AddObject(sm.NewMarker(*markers,
			color.RGBA{220, 0, 0, 255},
			16,
		))
	}

	img, err := ctx.Render()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=15")
	return png.Encode(w, img)
}

func scanTripShape(rows *sql.Rows) ([]MapPoint, error) {
	defer rows.Close()

	var out []MapPoint
	for rows.Next() {
		var p MapPoint
		if err := rows.Scan(&p.Lat, &p.Lon, &p.Seq); err != nil {
			return nil, err
		}
		out = append(out, p)
	}

	return out, rows.Err()
}

func TripMapHandler(s Server, w http.ResponseWriter, req *http.Request, params map[string]string) {
	rows, err := s.db.Query(tripMapSQL, params["trip_id"])
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shape, err := scanTripShape(rows)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(shape) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeMapPNG(w, 0, nil, shape)
}

type VehicleMapRow struct {
	RealtimeTripID string
	Status         string
	RdX            *int
	RdY            *int
}

func VehicleMapHandler(s Server, w http.ResponseWriter, req *http.Request, params map[string]string) {
	vehicleNumber, err := strconv.Atoi(params["vehicle_number"])
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var row VehicleMapRow

	err = s.db.QueryRow(vehicleMapSQL, params["data_owner"], vehicleNumber).Scan(
		new(any), /* operating_day, unused */
		new(any), /* data_owner_code, unused */
		new(any), /* vehicle_number, unused */
		&row.RealtimeTripID,
		&row.Status,
		new(any), /* event_timestamp, unused */
		&row.RdX,
		&row.RdY,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if row.RdX == nil || row.RdY == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	lat, lon := rijksdriehoek.RDtoWGS84(float64(*row.RdX), float64(*row.RdY))
	marker := s2.LatLngFromDegrees(lat, lon)

	var shape []MapPoint
	if row.Status != "END" && row.RealtimeTripID != "" {
		shapeRows, err := s.db.Query(vehicleTripShapeSQL, row.RealtimeTripID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		shape, err = scanTripShape(shapeRows)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	writeMapPNG(w, 14, &marker, shape)
}
