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

SELECT DISTINCT ON (
	e.stop_sequence,
    e.mode
)
	e.mode::text,
	e.stop_sequence,
	e.stop_id,
	e.stop_code,
	e.stop_name,
	e.platform_code,

	EXTRACT(EPOCH FROM e.scheduled_time)::bigint AS scheduled_time,

	k.status,
    k.operating_day,
	k.vehicle_number,
	kt.block_code,
	k.punctuality
FROM active_gtfs_stop_events e
LEFT JOIN kv6_trip_stop_status k
	ON k.operating_day = current_date
   AND k.realtime_trip_id = e.realtime_trip_id
   AND k.user_stop_code = e.stop_code
   AND (
       (e.mode = 'arrival' AND k.status IN ('ARRIVAL', 'ONROUTE'))
     OR (e.mode = 'departure' AND k.status IN ('DEPARTURE', 'ONROUTE'))
)
LEFT JOIN kv6_current_trip kt 
	ON k.operating_day = current_date
   AND k.realtime_trip_id = e.realtime_trip_id
WHERE e.trip_id = $1 
    AND NOT e.terminal
ORDER BY
	e.stop_sequence,
    e.mode,
    k.status = 'ONROUTE';

