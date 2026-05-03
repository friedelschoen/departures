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

WITH active_feed AS (
	SELECT id
	FROM gtfs_feeds
	WHERE active = true
	ORDER BY imported_at DESC
	LIMIT 1
)
SELECT 
	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

	t.trip_id,
	t.trip_headsign AS headsign,
	s.platform_code,

	EXTRACT(EPOCH FROM (
		(cd.date::timestamp + st.departure_time)
			AT TIME ZONE a.agency_timezone
	))::bigint AS scheduled_time,

	k.status,
    EXTRACT(EPOCH FROM k.timestamp)::bigint,
    k.punctuality,
	k.vehicle_number,
	k.block_code,
	k.rd_x,
	k.rd_y
	
FROM kv6_journey_status k
JOIN active_feed af ON true
JOIN gtfs_stops s
	ON s.feed_ref = af.id
   AND s.stop_code = k.user_stop_code
JOIN gtfs_trips t
	ON t.feed_ref = af.id
   AND t.realtime_trip_id = k.journey_key
   AND t.realtime_trip_sequence = 1
JOIN gtfs_stop_times st
    ON st.feed_ref = s.feed_ref
   AND st.trip_id = t.trip_id
   AND st.stop_id = s.stop_id
JOIN gtfs_calendar_dates cd
	ON cd.feed_ref = t.feed_ref
   AND cd.service_id = t.service_id
   AND cd.date = k.operating_day
   AND cd.exception_type = 1
JOIN gtfs_routes r
	ON r.feed_ref = t.feed_ref
   AND r.route_id = t.route_id
JOIN gtfs_agency a
	ON a.feed_ref = r.feed_ref
   AND a.agency_id = r.agency_id
WHERE k.vehicle_number IS NOT NULL
  AND (s.stop_id = $1 OR s.parent_station = $1)
  AND k.status IN ('ARRIVAL', 'ONSTOP', 'INIT')
  AND k.timestamp > now() - interval '5 minutes'
ORDER BY 
    (cd.date::timestamp + st.departure_time)
		AT TIME ZONE a.agency_timezone ASC, k.vehicle_number;
