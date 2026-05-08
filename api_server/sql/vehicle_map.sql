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
	operating_day,
	data_owner_code,
	vehicle_number,
	realtime_trip_id,
	status,
	event_timestamp,
	rd_x,
	rd_y
FROM kv6_current_vehicle
WHERE data_owner_code = $1
  AND vehicle_number = $2
ORDER BY operating_day DESC, event_timestamp DESC
LIMIT 1;
