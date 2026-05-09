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
	k.vehicle_number,
	k.status,
	k.rd_x,
	k.rd_y
FROM active_gtfs_trips t
JOIN kv6_current_trip k
	ON k.realtime_trip_id = t.realtime_trip_id
WHERE t.trip_id = $1
  AND k.status <> 'END'
  AND k.rd_x IS NOT NULL
  AND k.rd_y IS NOT NULL
ORDER BY k.event_timestamp DESC
LIMIT 1;
