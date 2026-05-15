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

//go:embed sql/stop_routes.sql
var stopRoutesSQL string

//go:embed sql/stop_vehicles.sql
var stopVehiclesSQL string

//go:embed sql/trip_info.sql
var tripInfoSQL string

//go:embed sql/trip_shape.sql
var tripShapeSQL string

//go:embed sql/trip_stops.sql
var tripStopsSQL string

//go:embed sql/trip_context.sql
var tripContextSQL string

//go:embed sql/vehicle_info.sql
var vehicleInfoSQL string

//go:embed sql/query.sql
var querySQL string

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
	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/stop/:stop",
		Title:    "Stop",
		Query:    stopInfoSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["stop"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/stop/:stop/arrivals",
		Title:    "StopArrival",
		Query:    stopDepartureSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{"arrival", "", "1 hour", "1 hour"}, nil
			}
			from := intervalParam(req, "from", "5 minutes")
			to := intervalParam(req, "to", "1 hours")

			return []any{"arrival", params["stop"], from, to}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/stop/:stop/departures",
		Title:    "StopDeparture",
		Query:    stopDepartureSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{"departure", "", "1 hour", "1 hour"}, nil
			}
			from := intervalParam(req, "from", "5 minutes")
			to := intervalParam(req, "to", "2 hours")

			return []any{"departure", params["stop"], from, to}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/stop/:stop/vehicles",
		Title:    "StopVehicle",
		Query:    stopVehiclesSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["stop"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/stop/:stop/routes",
		Title:    "StopRoute",
		Query:    stopRoutesSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["stop"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/stop_query",
		Title:    "StopQuery",
		Query:    stopQuerySQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			q := req.URL.Query().Get("q")
			return []any{q}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/search",
		Title:    "Query",
		Query:    querySQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			q := req.URL.Query().Get("q")
			return []any{q}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/vehicle/:data_owner/:vehicle_number",
		Title:    "Vehicle",
		Query:    vehicleInfoSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{"", "0"}, nil
			}
			return []any{
				params["data_owner"],
				params["vehicle_number"],
			}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/trip/:trip_id",
		Title:    "Trip",
		Query:    tripInfoSQL,
		Single:   true,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["trip_id"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/trip/:trip_id/shape",
		Title:    "TripShape",
		Query:    tripShapeSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["trip_id"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/trip/:trip_id/stops",
		Title:    "TripStop",
		Query:    tripStopsSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["trip_id"]}, nil
		},
	},
	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/trip/:trip_id/context",
		Title:    "TripStop",
		Query:    tripContextSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{""}, nil
			}
			return []any{params["trip_id"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/block/:data_owner/:block_code",
		Title:    "Block",
		Query:    blockInfoSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{"", "0"}, nil
			}
			return []any{params["data_owner"], params["block_code"]}, nil
		},
	},

	SQLHandler{
		Methods:  []string{"GET"},
		Endpoint: "/api/route/:route_id/trips",
		Title:    "RouteTrip",
		Query:    routeTripsSQL,
		ArgsFn: func(req *http.Request, params map[string]string) ([]any, error) {
			if params == nil {
				return []any{"", "1 hour", "1 hour"}, nil
			}
			from := intervalParam(req, "from", "30 minutes")
			to := intervalParam(req, "to", "6 hours")

			return []any{params["route_id"], from, to}, nil
		},
	},
}
