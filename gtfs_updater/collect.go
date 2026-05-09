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

func collectRoutes(a string, agencies map[string]struct{}) (routes map[string]struct{}, _ error) {
	routes = make(map[string]struct{})
	return routes, iterCSV(a, "routes.txt", func(row map[string]string) error {
		agencyID := row["agency_id"]
		if _, ok := agencies[agencyID]; !ok {
			return nil
		}
		routeID := row["route_id"]
		routes[routeID] = struct{}{}
		return nil
	})
}

func collectStops(a string, stops map[string]struct{}) error {
	stations := make(map[string]struct{})
	err := iterCSV(a, "stops.txt", func(row map[string]string) error {
		stopID := row["stop_id"]
		if _, ok := stops[stopID]; !ok {
			return nil
		}
		parentStation := row["parent_station"]
		if parentStation != "" {
			stations[parentStation] = struct{}{}
		}
		return nil
	})
	for stop := range stations {
		stops[stop] = struct{}{}
	}
	return err
}

func collectTrips(a string, routes map[string]struct{}) (trips, services, shapes map[string]struct{}, _ error) {
	trips = make(map[string]struct{})
	services = make(map[string]struct{})
	shapes = make(map[string]struct{})
	return trips, services, shapes, iterCSV(a, "trips.txt", func(row map[string]string) error {
		routeID := row["route_id"]
		if _, ok := routes[routeID]; !ok {
			return nil
		}
		tripID := row["trip_id"]
		trips[tripID] = struct{}{}
		serviceID := row["service_id"]
		services[serviceID] = struct{}{}
		shapeID := row["shape_id"]
		if shapeID != "" {
			shapes[shapeID] = struct{}{}
		}
		return nil
	})
}

func collectStopTimes(a string, trips map[string]struct{}) (stops map[string]struct{}, _ error) {
	stops = make(map[string]struct{})
	return stops, iterCSV(a, "stop_times.txt", func(row map[string]string) error {
		tripID := row["trip_id"]
		if _, ok := trips[tripID]; !ok {
			return nil
		}
		stopID := row["stop_id"]
		stops[stopID] = struct{}{}
		return nil
	})
}
