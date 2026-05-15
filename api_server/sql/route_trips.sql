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
	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

	t.trip_id,
	t.realtime_trip_id,
	t.trip_headsign,
	t.trip_short_name,
	t.direction_id,
	t.block_id,

	EXTRACT(EPOCH FROM (cd.date::timestamp + tb.start_time))::bigint AS start_time,
	EXTRACT(EPOCH FROM (cd.date::timestamp + tb.end_time))::bigint AS end_time,

	tb.start_stop,
	tb.end_stop,
	ss.stop_name AS start_stop_name,
	es.stop_name AS end_stop_name,

	k.status,
    k.operating_day,
	EXTRACT(EPOCH FROM k.event_timestamp)::bigint AS last_seen,
	COALESCE(k.punctuality, 0) AS punctuality,
	k.vehicle_number,
	k.block_code,
	k.rd_x,
	k.rd_y,
    k.lat,
    k.lon
FROM active_gtfs_routes r
JOIN active_gtfs_trips t
    ON t.route_id = r.route_id
JOIN active_gtfs_calendar_dates cd
    ON cd.service_id = t.service_id
   AND cd.exception_type = 1
JOIN active_gtfs_trip_bounds tb
    ON tb.trip_id = t.trip_id
LEFT JOIN active_gtfs_stops ss
    ON ss.stop_id = tb.start_stop
LEFT JOIN active_gtfs_stops es
    ON es.stop_id = tb.end_stop
LEFT JOIN kv6_current_trip k
	ON k.operating_day = cd.date
   AND k.realtime_trip_id = t.realtime_trip_id
WHERE r.route_id = $1
  AND (cd.date::timestamp + tb.start_time)
		BETWEEN now() - $2::interval
		    AND now() + $3::interval
ORDER BY
	(cd.date::timestamp + tb.start_time)
	+ (COALESCE(k.punctuality, 0) * interval '1 second');
