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

WITH stop_scope AS (
	SELECT
		s.feed_ref,
		s.stop_id,
		s.stop_code,
		s.stop_name,
		s.platform_code
	FROM active_gtfs_stops s
	WHERE s.stop_id = $1
	   OR s.parent_station = $1
),
vehicle AS (
	SELECT *
	FROM kv6_current_vehicle v
	WHERE v.event_timestamp > now() - interval '5 minutes'
	  AND v.status IN ('ARRIVAL', 'ONSTOP', 'INIT')
)
SELECT
	v.operating_day,
	v.data_owner_code,
	v.vehicle_number,
	v.status,
	EXTRACT(EPOCH FROM v.event_timestamp)::bigint AS last_seen,
	v.user_stop_code,
	v.passage_sequence_number,
	v.realtime_trip_id,
	v.block_code,
	COALESCE(v.punctuality, 0) AS punctuality,
	v.rd_x,
	v.rd_y,
    v.lat,
    v.lon,

	s.stop_id,
	s.stop_name,
	s.platform_code,

	t.trip_id,
	t.trip_headsign,
	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color
FROM vehicle v
JOIN stop_scope s
	ON s.stop_code = v.user_stop_code
LEFT JOIN active_gtfs_trips t
	ON t.feed_ref = s.feed_ref
   AND t.realtime_trip_id = v.realtime_trip_id
   AND t.realtime_trip_sequence = 1
LEFT JOIN active_gtfs_calendar_dates cd
	ON cd.feed_ref = t.feed_ref
   AND cd.service_id = t.service_id
   AND cd.date = v.operating_day
   AND cd.exception_type = 1
LEFT JOIN active_gtfs_routes r
	ON r.feed_ref = t.feed_ref
   AND r.route_id = t.route_id
WHERE t.trip_id IS NULL
   OR cd.service_id IS NOT NULL
ORDER BY
    v.punctuality DESC,
	v.vehicle_number,
	v.event_timestamp DESC;
