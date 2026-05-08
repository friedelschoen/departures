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
	h.operating_day,
	h.data_owner_code,
	h.block_code,
	h.realtime_trip_id,
	h.line_planning_number,
	h.trip_short_name,
	h.reinforcement_number,
	EXTRACT(EPOCH FROM h.first_seen)::bigint AS first_seen,
	EXTRACT(EPOCH FROM h.last_seen)::bigint AS last_seen,

	t.trip_id,
	t.trip_headsign,
	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

	v.vehicle_number,
	v.status,
	EXTRACT(EPOCH FROM v.event_timestamp)::bigint AS vehicle_last_seen,
	v.rd_x,
	v.rd_y 
FROM kv6_block_trip_history h
LEFT JOIN active_gtfs_trips t
    ON t.realtime_trip_id = h.realtime_trip_id
   AND t.realtime_trip_sequence = 1
LEFT JOIN active_gtfs_routes r
    ON r.route_id = t.route_id
LEFT JOIN kv6_current_vehicle v
	ON v.operating_day = h.operating_day
   AND v.data_owner_code = h.data_owner_code
   AND v.block_code = h.block_code
WHERE h.block_code = $1
ORDER BY h.operating_day DESC, h.first_seen DESC;
