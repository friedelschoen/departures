CREATE INDEX IF NOT EXISTS idx_stop_events_lookup
ON gtfs_stop_events (
	stop_id,
	mode,
	scheduled_time
);

CREATE INDEX IF NOT EXISTS idx_stops_parent_station
ON gtfs_stops (
	parent_station
);

CREATE INDEX IF NOT EXISTS idx_kv6_current_trip_lookup
ON kv6_current_trip (
	operating_day,
	realtime_trip_id
);

CREATE INDEX IF NOT EXISTS idx_kv6_trip_stop_status_lookup
ON kv6_trip_stop_status (
	operating_day,
	realtime_trip_id,
	user_stop_code
);

CREATE INDEX IF NOT EXISTS idx_gtfs_trips_route_service
ON gtfs_trips (
	route_id,
	service_id
);

CREATE INDEX IF NOT EXISTS idx_calendar_dates_active
ON gtfs_calendar_dates (
	service_id,
	date
)
WHERE exception_type = 1;

CREATE INDEX IF NOT EXISTS idx_stop_events_routes
ON gtfs_stop_events (
	stop_id,
	scheduled_time,
	route_id
);

CREATE INDEX IF NOT EXISTS idx_kv6_block_history_lookup
ON kv6_block_trip_history (
	operating_day,
	block_code,
	first_seen
);

CREATE INDEX IF NOT EXISTS idx_kv6_current_trip_block
ON kv6_current_trip (
	operating_day,
	block_code
);

CREATE INDEX IF NOT EXISTS idx_gtfs_trips_realtime
ON gtfs_trips (
	realtime_trip_id
);

CREATE INDEX IF NOT EXISTS idx_stop_events_realtime
ON gtfs_stop_events (
	realtime_trip_id
);
