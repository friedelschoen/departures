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

import "database/sql"

func importAgency(tx *sql.Tx, feedRef int64, a string, agencies map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_agency (
				feed_ref,
				agency_id,
				agency_name,
				agency_url,
				agency_timezone,
				agency_phone
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "agency.txt", func(row map[string]string) []any {
		agencyID := row["agency_id"]
		if _, ok := agencies[agencyID]; !ok {
			return nil
		}
		return []any{
			feedRef,
			row["agency_id"],
			row["agency_name"],
			nullString(row["agency_url"]),
			nullString(row["agency_timezone"]),
			nullString(row["agency_phone"]),
		}
	}, nil)
}

func importCalendarDates(tx *sql.Tx, feedRef int64, a string, services map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_calendar_dates (
				feed_ref,
				service_id,
				date,
				exception_type
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "calendar_dates.txt", func(row map[string]string) []any {
		serviceID := row["service_id"]
		if _, ok := services[serviceID]; !ok {
			return nil
		}
		return []any{
			feedRef,
			row["service_id"],
			parseGTFSDate(row["date"]),
			parseInt(row["exception_type"]),
		}
	}, nil)
}

func importRoutes(tx *sql.Tx, feedRef int64, a string, routes map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_routes (
				feed_ref,
				route_id,
				agency_id,
				route_short_name,
				route_long_name,
				route_desc,
				route_type,
				route_color,
				route_text_color,
				route_url
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "routes.txt", func(row map[string]string) []any {
		routeID := row["route_id"]
		if _, ok := routes[routeID]; !ok {
			return nil
		}

		return []any{
			feedRef,
			routeID,
			row["agency_id"],
			nullString(row["route_short_name"]),
			nullString(row["route_long_name"]),
			nullString(row["route_desc"]),
			nullString(row["route_type"]),
			nullString(row["route_color"]),
			nullString(row["route_text_color"]),
			nullString(row["route_url"]),
		}
	}, nil)
}

func importShapes(tx *sql.Tx, feedRef int64, a string, shapes map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_shapes (
				feed_ref,
				shape_id,
				shape_pt_sequence,
				shape_pt_lat,
				shape_pt_lon,
				shape_dist_traveled
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "shapes.txt", func(row map[string]string) []any {
		shapeID := row["shape_id"]
		if _, ok := shapes[shapeID]; !ok {
			return nil
		}
		return []any{
			feedRef,
			row["shape_id"],
			parseInt(row["shape_pt_sequence"]),
			parseFloat(row["shape_pt_lat"]),
			parseFloat(row["shape_pt_lon"]),
			parseNullableFloat(row["shape_dist_traveled"]),
		}
	}, nil)
}

func importStops(tx *sql.Tx, feedRef int64, a string, stops map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_stops (
				feed_ref,
				stop_id,
				stop_code,
				stop_name,
				stop_lat,
				stop_lon,
				location_type,
				parent_station,
				stop_timezone,
				wheelchair_boarding,
				platform_code,
				zone_id
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "stops.txt", func(row map[string]string) []any {
		stopID := row["stop_id"]
		if _, ok := stops[stopID]; !ok {
			return nil
		}
		return []any{
			feedRef,
			stopID,
			nullString(row["stop_code"]),
			nullString(row["stop_name"]),
			parseNullableFloat(row["stop_lat"]),
			parseNullableFloat(row["stop_lon"]),
			parseNullableInt(row["location_type"]),
			nullString(row["parent_station"]),
			nullString(row["stop_timezone"]),
			parseNullableInt(row["wheelchair_boarding"]),
			nullString(row["platform_code"]),
			nullString(row["zone_id"]),
		}
	}, nil)
}

func importTrips(tx *sql.Tx, feedRef int64, a string, routes map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_trips (
				feed_ref,
				route_id,
				service_id,
				trip_id,
				realtime_trip_id,
				trip_headsign,
				trip_short_name,
				trip_long_name,
				direction_id,
				block_id,
				shape_id,
				wheelchair_accessible,
				bikes_allowed
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "trips.txt", func(row map[string]string) []any {
		routeID := row["route_id"]
		if _, ok := routes[routeID]; !ok {
			return nil
		}
		return []any{
			feedRef,
			routeID,
			row["service_id"],
			row["trip_id"],
			nullString(row["realtime_trip_id"]),
			nullString(row["trip_headsign"]),
			nullString(row["trip_short_name"]),
			nullString(row["trip_long_name"]),
			parseNullableInt(row["direction_id"]),
			nullString(row["block_id"]),
			nullString(row["shape_id"]),
			parseNullableInt(row["wheelchair_accessible"]),
			parseNullableInt(row["bikes_allowed"]),
		}
	}, nil)
}

func importStopTimes(tx *sql.Tx, feedRef int64, a string, trips map[string]struct{}) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_stop_times (
				feed_ref,
				trip_id,
				stop_sequence,
				stop_id,
				stop_headsign,
				arrival_time,
				departure_time,
				pickup_type,
				drop_off_type,
				timepoint,
				shape_dist_traveled,
				fare_units_traveled
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "stop_times.txt", func(row map[string]string) []any {
		tripID := row["trip_id"]
		if _, ok := trips[tripID]; !ok {
			return nil
		}
		return []any{
			feedRef,
			tripID,
			parseInt(row["stop_sequence"]),
			row["stop_id"],
			nullString(row["stop_headsign"]),
			nullString(row["arrival_time"]),
			nullString(row["departure_time"]),
			parseNullableInt(row["pickup_type"]),
			parseNullableInt(row["drop_off_type"]),
			parseNullableInt(row["timepoint"]),
			parseNullableFloat(row["shape_dist_traveled"]),
			parseNullableInt(row["fare_units_traveled"]),
		}
	}, nil)
}
