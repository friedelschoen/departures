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
	st.stop_sequence,
	st.stop_id,
	s.stop_code,
	s.stop_name,
	s.platform_code,
	st.arrival_time,
	st.departure_time,

	k.status,
	EXTRACT(EPOCH FROM k.event_timestamp)::bigint AS last_seen,
	k.vehicle_number,
	k.block_code,
	k.punctuality
FROM active_gtfs_trips t
JOIN active_gtfs_stop_times st
    ON st.trip_id = t.trip_id
JOIN active_gtfs_stops s
    ON s.stop_id = st.stop_id
LEFT JOIN kv6_trip_stop_status k
	ON k.operating_day = current_date
   AND k.realtime_trip_id = t.realtime_trip_id
   AND k.user_stop_code = s.stop_code
WHERE t.trip_id = $1
ORDER BY st.stop_sequence;
