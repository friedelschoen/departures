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

SELECT
	t.trip_id,
	t.realtime_trip_id,
	t.trip_headsign,
	t.trip_short_name,
	t.direction_id,
	t.block_id,

	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

    EXTRACT(EPOCH FROM tb.start_time)::bigint as start_time,
    EXTRACT(EPOCH FROM tb.end_time)::bigint as end_time,
	tb.start_stop,
	tb.end_stop,

	k.status,
    k.operating_day,
	EXTRACT(EPOCH FROM k.event_timestamp)::bigint AS last_seen,
	k.vehicle_number,
    k.data_owner_code,
	k.block_code,
	k.punctuality,
	k.rd_x,
	k.rd_y,
    k.lat,
    k.lon
FROM active_gtfs_trips t
JOIN active_gtfs_routes r
    ON r.route_id = t.route_id
LEFT JOIN active_gtfs_trip_bounds tb
    ON tb.trip_id = t.trip_id
LEFT JOIN kv6_current_trip k
	ON k.realtime_trip_id = t.realtime_trip_id
WHERE t.trip_id = $1;
