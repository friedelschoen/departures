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
	e.mode::text,

	e.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

	e.trip_id,

	COALESCE(
		NULLIF(e.stop_headsign, ''),
		t.trip_headsign
	) AS headsign,

	s.platform_code,

	EXTRACT(EPOCH FROM e.scheduled_time)::bigint AS scheduled_time,

	e.terminal,

	k.status,
    k.operating_day,
	EXTRACT(EPOCH FROM k.event_timestamp)::bigint AS last_seen,

	CASE 
		WHEN e.first_stop
		THEN GREATEST(COALESCE(k.punctuality, 0), 0)
		ELSE COALESCE(k.punctuality, 0)
	END as punctuality,

	k.vehicle_number,
	k.block_code,
	k.rd_x,
	k.rd_y,
    k.lat,
    k.lon,
    k.user_stop_code = e.stop_code as at_stop,

    (
        e.mode = 'departure'
        AND e.scheduled_time BETWEEN now()
            AND now() + interval '5 minutes'
        AND k.vehicle_number IS NULL
        AND NOT e.terminal
    ) AS warning
FROM active_gtfs_stop_events e
JOIN active_gtfs_stops s
    ON s.stop_id = e.stop_id
   AND (
		s.stop_id = $2
		OR s.parent_station = $2
   )
JOIN active_gtfs_trips t
    ON t.trip_id = e.trip_id
JOIN active_gtfs_routes r
    ON r.route_id = e.route_id
LEFT JOIN kv6_current_trip k
	ON k.operating_day = e.service_date
   AND k.realtime_trip_id = t.realtime_trip_id
WHERE e.mode = $1::gtfs_stop_event_mode
  AND e.scheduled_time BETWEEN
		now() - $3::interval
		AND
		now() + $4::interval
ORDER BY
	e.scheduled_time;
