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

CREATE TABLE IF NOT EXISTS gtfs_feeds (
	id BIGSERIAL PRIMARY KEY,

	feed_publisher_name TEXT,
	feed_id TEXT,
	feed_publisher_url TEXT,
	feed_lang TEXT,
	feed_start_date DATE,
	feed_end_date DATE,
	feed_version TEXT,

	active BOOLEAN NOT NULL DEFAULT false,
	imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),

	UNIQUE (feed_id, feed_version)
);

CREATE TABLE IF NOT EXISTS gtfs_agency (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	agency_id TEXT NOT NULL,
	agency_name TEXT NOT NULL,
	agency_url TEXT,
	agency_timezone TEXT,
	agency_phone TEXT,

	PRIMARY KEY (feed_ref, agency_id)
);

CREATE TABLE IF NOT EXISTS gtfs_calendar_dates (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	service_id TEXT NOT NULL,
	date DATE NOT NULL,
	exception_type SMALLINT NOT NULL,

	PRIMARY KEY (feed_ref, service_id, date)
);

CREATE TABLE IF NOT EXISTS gtfs_routes (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	route_id TEXT NOT NULL,
	agency_id TEXT,
	route_short_name TEXT,
	route_long_name TEXT,
	route_desc TEXT,
	route_type SMALLINT,
	route_color TEXT,
	route_text_color TEXT,
	route_url TEXT,

	PRIMARY KEY (feed_ref, route_id),

	FOREIGN KEY (feed_ref, agency_id)
		REFERENCES gtfs_agency(feed_ref, agency_id)
		ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS gtfs_shapes (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	shape_id TEXT NOT NULL,
	shape_pt_sequence INTEGER NOT NULL,
	shape_pt_lat DOUBLE PRECISION NOT NULL,
	shape_pt_lon DOUBLE PRECISION NOT NULL,
	shape_dist_traveled DOUBLE PRECISION,
    
    PRIMARY KEY (feed_ref, shape_id, shape_pt_sequence)
);

CREATE TABLE IF NOT EXISTS gtfs_stops (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	stop_id TEXT NOT NULL,
	stop_code TEXT,
	stop_name TEXT,
	stop_lat DOUBLE PRECISION,
	stop_lon DOUBLE PRECISION,
	location_type SMALLINT,
	parent_station TEXT,
	stop_timezone TEXT,
	wheelchair_boarding SMALLINT,
	platform_code TEXT,
	zone_id TEXT,

	PRIMARY KEY (feed_ref, stop_id),

	FOREIGN KEY (feed_ref, parent_station)
		REFERENCES gtfs_stops(feed_ref, stop_id)
		ON DELETE SET NULL
		DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE IF NOT EXISTS gtfs_trips (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	route_id TEXT NOT NULL,
	service_id TEXT NOT NULL,
	trip_id TEXT NOT NULL,
	realtime_trip_id TEXT,
    realtime_trip_sequence INTEGER,
	trip_headsign TEXT,
	trip_short_name TEXT,
	trip_long_name TEXT,
	direction_id SMALLINT,
	block_id TEXT,
	shape_id TEXT,
	wheelchair_accessible SMALLINT,
	bikes_allowed SMALLINT,

	PRIMARY KEY (feed_ref, trip_id),

	FOREIGN KEY (feed_ref, route_id)
		REFERENCES gtfs_routes(feed_ref, route_id)
		ON DELETE CASCADE,

    UNIQUE (feed_ref, service_id, realtime_trip_id, realtime_trip_sequence)
);

CREATE TABLE IF NOT EXISTS gtfs_stop_times (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	trip_id TEXT NOT NULL,
	stop_sequence INTEGER NOT NULL,
	stop_id TEXT NOT NULL,
	stop_headsign TEXT,
	arrival_time INTERVAL,
	departure_time INTERVAL,
	pickup_type SMALLINT NOT NULL DEFAULT 0,
	drop_off_type SMALLINT NOT NULL DEFAULT 0,
	timepoint SMALLINT,
	shape_dist_traveled DOUBLE PRECISION,
	fare_units_traveled INTEGER,

	PRIMARY KEY (feed_ref, trip_id, stop_sequence),

	FOREIGN KEY (feed_ref, trip_id)
		REFERENCES gtfs_trips(feed_ref, trip_id)
		ON DELETE CASCADE,

	FOREIGN KEY (feed_ref, stop_id)
		REFERENCES gtfs_stops(feed_ref, stop_id)
		ON DELETE CASCADE
);

CREATE TYPE gtfs_stop_event_mode AS ENUM (
	'arrival',
	'departure'
);

CREATE UNLOGGED TABLE IF NOT EXISTS gtfs_stop_events (
	feed_ref BIGINT NOT NULL REFERENCES gtfs_feeds(id) ON DELETE CASCADE,

	mode gtfs_stop_event_mode NOT NULL,

	service_id TEXT NOT NULL,
	service_date DATE NOT NULL,

	trip_id TEXT NOT NULL,
    direction_id SMALLINT,
	realtime_trip_id TEXT,
	realtime_trip_sequence INTEGER,

	route_id TEXT NOT NULL,
	route_short_name TEXT,
	route_color TEXT,
	route_text_color TEXT,

	stop_sequence INTEGER NOT NULL,
	stop_id TEXT NOT NULL,
	stop_code TEXT,
	stop_name TEXT,
	platform_code TEXT,

	stop_headsign TEXT,
	trip_headsign TEXT,

	scheduled_time TIMESTAMPTZ NOT NULL,

	terminal BOOLEAN NOT NULL,
	first_stop BOOLEAN NOT NULL,
	last_stop BOOLEAN NOT NULL,
	event_type SMALLINT NOT NULL DEFAULT 0,
	timepoint SMALLINT,
	shape_dist_traveled DOUBLE PRECISION,
	fare_units_traveled INTEGER,

	PRIMARY KEY (
		feed_ref,
		mode,
		service_date,
		trip_id,
		stop_sequence
	),

	FOREIGN KEY (feed_ref, trip_id)
		REFERENCES gtfs_trips(feed_ref, trip_id)
		ON DELETE CASCADE,

	FOREIGN KEY (feed_ref, stop_id)
		REFERENCES gtfs_stops(feed_ref, stop_id)
		ON DELETE CASCADE,

	FOREIGN KEY (feed_ref, trip_id, stop_sequence)
		REFERENCES gtfs_stop_times(feed_ref, trip_id, stop_sequence)
		ON DELETE CASCADE
);

CREATE UNLOGGED TABLE IF NOT EXISTS gtfs_trip_bounds (
	feed_ref BIGINT NOT NULL,

	trip_id TEXT NOT NULL,
	start_time INTERVAL NOT NULL,
	end_time INTERVAL,
	start_sequence INTEGER NOT NULL,
	end_sequence INTEGER NOT NULL,
	start_stop TEXT NOT NULL,
	end_stop TEXT NOT NULL,

	PRIMARY KEY (feed_ref, trip_id),

	FOREIGN KEY (feed_ref, trip_id)
		REFERENCES gtfs_trips(feed_ref, trip_id)
		ON DELETE CASCADE,

	FOREIGN KEY (feed_ref, trip_id, start_sequence)
		REFERENCES gtfs_stop_times(feed_ref, trip_id, stop_sequence)
		ON DELETE CASCADE,

	FOREIGN KEY (feed_ref, trip_id, end_sequence)
		REFERENCES gtfs_stop_times(feed_ref, trip_id, stop_sequence)
		ON DELETE CASCADE,

	FOREIGN KEY (feed_ref, start_stop)
		REFERENCES gtfs_stops(feed_ref, stop_id)
		ON DELETE CASCADE,

    FOREIGN KEY (feed_ref, end_stop)
		REFERENCES gtfs_stops(feed_ref, stop_id)
		ON DELETE CASCADE
);

