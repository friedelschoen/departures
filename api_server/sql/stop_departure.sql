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

SELECT DISTINCT ON (k.last_event_id)
	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

	t.trip_id,
	COALESCE(NULLIF(st.stop_headsign, ''), t.trip_headsign) AS headsign,
	s.platform_code,

	EXTRACT(EPOCH FROM (
		(cd.date::timestamp + CASE WHEN $1 = 'departure' THEN st.departure_time ELSE st.arrival_time END)
			AT TIME ZONE a.agency_timezone
	))::bigint AS scheduled_time,

	st.stop_sequence = tb.end_sequence AS terminal,

	k.status,
	EXTRACT(EPOCH FROM k.event_timestamp)::bigint AS last_seen,
	COALESCE(k.punctuality, 0) AS punctuality,
	k.vehicle_number,
	k.block_code,
	k.rd_x,
	k.rd_y,

	CASE
		WHEN
			((cd.date::timestamp + st.departure_time) AT TIME ZONE a.agency_timezone)
				BETWEEN now() AND now() + interval '5 minutes'
			AND k.vehicle_number IS NULL
			AND st.stop_sequence <> tb.end_sequence
		THEN true
		ELSE false
	END AS warning
FROM active_gtfs_stop_times st
JOIN active_gtfs_stops s
    ON s.stop_id = st.stop_id
   AND (s.stop_id = $2 OR s.parent_station = $2)
JOIN active_gtfs_trips t
    ON t.trip_id = st.trip_id
JOIN active_gtfs_routes r
    ON r.route_id = t.route_id
JOIN active_gtfs_agency a
    ON a.agency_id = r.agency_id
JOIN active_gtfs_calendar_dates cd
    ON cd.service_id = t.service_id
   AND cd.exception_type = 1
   AND cd.date BETWEEN
		((now() AT TIME ZONE a.agency_timezone)::date - 1)
		AND
		((now() AT TIME ZONE a.agency_timezone)::date + 1)
JOIN active_gtfs_trip_bounds tb
    ON tb.trip_id = st.trip_id
JOIN kv6_current_trip k
	ON k.operating_day = cd.date
   AND k.realtime_trip_id = t.realtime_trip_id
WHERE (
	(cd.date::timestamp + st.departure_time)
		AT TIME ZONE a.agency_timezone
) BETWEEN now() - $3::interval
    AND now() + $4::interval
ORDER BY k.last_event_id,
	((cd.date::timestamp + st.departure_time) AT TIME ZONE a.agency_timezone)
	+ (COALESCE(k.punctuality, 0) * interval '1 second');
