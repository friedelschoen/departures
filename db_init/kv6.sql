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

CREATE TABLE IF NOT EXISTS kv6_events (
	id BIGSERIAL PRIMARY KEY,
	received_at TIMESTAMPTZ NOT NULL DEFAULT now(),

	operating_day DATE NOT NULL,
	data_owner_code TEXT NOT NULL,
	line_planning_number TEXT NOT NULL,
	trip_short_name INTEGER NOT NULL,
	reinforcement_number INTEGER NOT NULL DEFAULT 0,
	realtime_trip_id TEXT NOT NULL,

	status TEXT NOT NULL,
	event_timestamp TIMESTAMPTZ NOT NULL,
	source TEXT NOT NULL,

	user_stop_code TEXT,
	passage_sequence_number INTEGER,

	vehicle_number INTEGER,
	block_code INTEGER,
   
    wheelchair_accessible BOOLEAN, -- ACCESSIBLE,NOTACCESSIBLE,UNKNOWN -> true,false,null
    number_of_coaches INTEGER,

    punctuality INTEGER,
	rd_x INTEGER,
	rd_y INTEGER,
	distance_since_last_user_stop INTEGER
);

CREATE INDEX IF NOT EXISTS kv6_events_id_idx
ON kv6_events (id);

CREATE INDEX IF NOT EXISTS kv6_events_journey_time_idx
ON kv6_events (operating_day, realtime_trip_id, event_timestamp DESC);

CREATE INDEX IF NOT EXISTS kv6_events_vehicle_time_idx
ON kv6_events (operating_day, data_owner_code, vehicle_number, event_timestamp DESC)
WHERE vehicle_number IS NOT NULL;

CREATE INDEX IF NOT EXISTS kv6_events_block_time_idx
ON kv6_events (operating_day, data_owner_code, block_code, event_timestamp DESC)
WHERE block_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS kv6_events_stop_time_idx
ON kv6_events (user_stop_code, event_timestamp DESC)
WHERE user_stop_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS kv6_current_trip (
	operating_day DATE NOT NULL,
	data_owner_code TEXT NOT NULL,
	line_planning_number TEXT NOT NULL,
	trip_short_name INTEGER NOT NULL,
	reinforcement_number INTEGER NOT NULL DEFAULT 0,
	realtime_trip_id TEXT NOT NULL,

	status TEXT NOT NULL,
	event_timestamp TIMESTAMPTZ NOT NULL,

	vehicle_number INTEGER,
	block_code INTEGER,

	user_stop_code TEXT,
	passage_sequence_number INTEGER,

	punctuality INTEGER,
	rd_x INTEGER,
	rd_y INTEGER,

	last_event_id BIGINT NOT NULL,

	PRIMARY KEY (
		operating_day,
		data_owner_code,
		line_planning_number,
		trip_short_name,
		reinforcement_number
	)
);

CREATE UNIQUE INDEX IF NOT EXISTS kv6_current_trip_key_idx
ON kv6_current_trip (operating_day, realtime_trip_id);

CREATE INDEX IF NOT EXISTS kv6_current_trip_vehicle_idx
ON kv6_current_trip (operating_day, data_owner_code, vehicle_number)
WHERE vehicle_number IS NOT NULL;

CREATE INDEX IF NOT EXISTS kv6_current_trip_block_idx
ON kv6_current_trip (operating_day, data_owner_code, block_code)
WHERE block_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS kv6_current_vehicle (
	operating_day DATE NOT NULL,
	data_owner_code TEXT NOT NULL,
	vehicle_number INTEGER NOT NULL,

	realtime_trip_id TEXT NOT NULL,
	line_planning_number TEXT NOT NULL,
	trip_short_name INTEGER NOT NULL,
	reinforcement_number INTEGER NOT NULL DEFAULT 0,

	status TEXT NOT NULL,
	event_timestamp TIMESTAMPTZ NOT NULL,

	user_stop_code TEXT,
	passage_sequence_number INTEGER,
	block_code INTEGER,
	punctuality INTEGER,

	rd_x INTEGER,
	rd_y INTEGER,

	last_event_id BIGINT NOT NULL,

	PRIMARY KEY (
		operating_day,
		data_owner_code,
		vehicle_number
	)
);

CREATE INDEX IF NOT EXISTS kv6_current_vehicle_trip_idx
ON kv6_current_vehicle (operating_day, realtime_trip_id);

CREATE INDEX IF NOT EXISTS kv6_current_vehicle_block_idx
ON kv6_current_vehicle (operating_day, data_owner_code, block_code)
WHERE block_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS kv6_trip_stop_status (
	operating_day DATE NOT NULL,
	data_owner_code TEXT NOT NULL,
	line_planning_number TEXT NOT NULL,
	trip_short_name INTEGER NOT NULL,
	reinforcement_number INTEGER NOT NULL DEFAULT 0,
	realtime_trip_id TEXT NOT NULL,

	user_stop_code TEXT NOT NULL,
	passage_sequence_number INTEGER NOT NULL,

	status TEXT NOT NULL,
	event_timestamp TIMESTAMPTZ NOT NULL,

	vehicle_number INTEGER,
	block_code INTEGER,
	punctuality INTEGER,

	rd_x INTEGER,
	rd_y INTEGER,

	last_event_id BIGINT NOT NULL,

	PRIMARY KEY (
		operating_day,
		data_owner_code,
		line_planning_number,
		trip_short_name,
		reinforcement_number,
		user_stop_code,
		passage_sequence_number
	)
);

CREATE INDEX IF NOT EXISTS kv6_trip_stop_status_trip_idx
ON kv6_trip_stop_status (operating_day, realtime_trip_id, passage_sequence_number);

CREATE INDEX IF NOT EXISTS kv6_trip_stop_status_stop_idx
ON kv6_trip_stop_status (user_stop_code, event_timestamp DESC);

CREATE TABLE IF NOT EXISTS kv6_vehicle_trip_history (
	operating_day DATE NOT NULL,
	data_owner_code TEXT NOT NULL,
	vehicle_number INTEGER NOT NULL,
	realtime_trip_id TEXT NOT NULL,

	line_planning_number TEXT NOT NULL,
	trip_short_name INTEGER NOT NULL,
	reinforcement_number INTEGER NOT NULL DEFAULT 0,

	block_code INTEGER,

	first_seen TIMESTAMPTZ NOT NULL,
	last_seen TIMESTAMPTZ NOT NULL,

	first_event_id BIGINT NOT NULL,
	last_event_id BIGINT NOT NULL,

	PRIMARY KEY (
		operating_day,
		data_owner_code,
		vehicle_number,
		realtime_trip_id
	)
);

CREATE INDEX IF NOT EXISTS kv6_vehicle_trip_history_vehicle_idx
ON kv6_vehicle_trip_history (
	operating_day,
	data_owner_code,
	vehicle_number,
	first_seen
);

CREATE INDEX IF NOT EXISTS kv6_vehicle_trip_history_block_idx
ON kv6_vehicle_trip_history (
	operating_day,
	data_owner_code,
	block_code,
	first_seen
)
WHERE block_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS kv6_block_trip_history (
	operating_day DATE NOT NULL,
	data_owner_code TEXT NOT NULL,
	block_code INTEGER NOT NULL,
	realtime_trip_id TEXT NOT NULL,

	line_planning_number TEXT NOT NULL,
	trip_short_name INTEGER NOT NULL,
	reinforcement_number INTEGER NOT NULL DEFAULT 0,

	first_seen TIMESTAMPTZ NOT NULL,
	last_seen TIMESTAMPTZ NOT NULL,

	first_event_id BIGINT NOT NULL,
	last_event_id BIGINT NOT NULL,

	PRIMARY KEY (
		operating_day,
		data_owner_code,
		block_code,
		realtime_trip_id
	)
);

CREATE INDEX IF NOT EXISTS kv6_block_trip_history_lookup_idx
ON kv6_block_trip_history (
	operating_day,
	data_owner_code,
	block_code,
	first_seen
);

CREATE TABLE IF NOT EXISTS kv6_projection_offsets (
	projector_name TEXT PRIMARY KEY,
	last_event_id BIGINT NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- OR REPLACE -> IF NOT EXISTS
CREATE OR REPLACE VIEW trip_stop_detail AS
SELECT
	st.feed_ref,
	st.trip_id,
	st.stop_sequence,
	st.stop_id,
	s.stop_code,
	s.stop_name,
	s.platform_code,
	st.arrival_time,
	st.departure_time,

	t.realtime_trip_id AS realtime_trip_id,

	k.status AS realtime_status,
	k.event_timestamp,
	k.vehicle_number,
	k.block_code,
	k.punctuality
FROM gtfs_stop_times st
JOIN gtfs_stops s
	ON s.feed_ref = st.feed_ref
   AND s.stop_id = st.stop_id
JOIN gtfs_trips t
	ON t.feed_ref = st.feed_ref
   AND t.trip_id = st.trip_id
LEFT JOIN kv6_trip_stop_status k
	ON k.realtime_trip_id = t.realtime_trip_id
   AND k.user_stop_code = s.stop_code;

