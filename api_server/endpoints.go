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

//go:embed sql/block_info.sql
var blockInfoSQL string

//go:embed sql/route_trips.sql
var routeTripsSQL string

//go:embed sql/stop_departure.sql
var stopDepartureSQL string

//go:embed sql/stop_info.sql
var stopInfoSQL string

//go:embed sql/stop_query.sql
var stopQuerySQL string

//go:embed sql/stop_vehicles.sql
var stopVehiclesSQL string

//go:embed sql/trip_info.sql
var tripInfoSQL string

//go:embed sql/trip_map.sql
var tripMapSQL string

//go:embed sql/trip_stops.sql
var tripStopsSQL string

//go:embed sql/vehicle_info.sql
var vehicleInfoSQL string

//go:embed sql/vehicle_map.sql
var vehicleMapSQL string

func intervalParam(req *http.Request, name string, def string) string {
	v := req.URL.Query().Get(name)
	switch v {
	case "":
		return def
	case "5 minutes", "15 minutes", "30 minutes", "1 hour", "2 hours", "4 hours", "6 hours":
		return v
	default:
		return def
	}
}

var mux APIHandleMux = []APIHandler{
	{[]string{"GET"}, "/api/stop/:stop", SQLHandler(stopInfoSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		return []any{params["stop"]}, nil
	})},

	{[]string{"GET"}, "/api/stop/:stop/arrivals", SQLHandler(stopDepartureSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		from := intervalParam(req, "from", "5 minutes")
		to := intervalParam(req, "to", "1 hours")

		return []any{"arrival", params["stop"], from, to}, nil
	})},
	{[]string{"GET"}, "/api/stop/:stop/departures", SQLHandler(stopDepartureSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		from := intervalParam(req, "from", "5 minutes")
		to := intervalParam(req, "to", "2 hours")

		return []any{"departure", params["stop"], from, to}, nil
	})},

	{[]string{"GET"}, "/api/stop/:stop/vehicles", SQLHandler(stopVehiclesSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		return []any{params["stop"]}, nil
	})},

	{[]string{"GET"}, "/api/stop_query", SQLHandler(stopQuerySQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		q := req.URL.Query().Get("q")
		return []any{q}, nil
	})},

	{[]string{"GET"}, "/api/vehicle/:data_owner/:vehicle_number", SQLHandler(vehicleInfoSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		return []any{
			params["data_owner"],
			params["vehicle_number"],
		}, nil
	})},

	{[]string{"GET"}, "/api/trip/:trip_id", SQLHandler(tripInfoSQL, true, func(req *http.Request, params map[string]string) ([]any, error) {
		return []any{params["trip_id"]}, nil
	})},

	{[]string{"GET"}, "/api/trip/:trip_id/stops", SQLHandler(tripStopsSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		return []any{params["trip_id"]}, nil
	})},

	{[]string{"GET"}, "/api/block/:block_code", SQLHandler(blockInfoSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		return []any{params["block_code"]}, nil
	})},

	{[]string{"GET"}, "/api/route/:route_id/trips", SQLHandler(routeTripsSQL, false, func(req *http.Request, params map[string]string) ([]any, error) {
		from := intervalParam(req, "from", "30 minutes")
		to := intervalParam(req, "to", "6 hours")

		return []any{params["route_id"], from, to}, nil
	})},

	{[]string{"GET"}, "/api/vehicle/:data_owner/:vehicle_number/map.png", VehicleMapHandler},
	{[]string{"GET"}, "/api/trip/:trip_id/map.png", TripMapHandler},
}
