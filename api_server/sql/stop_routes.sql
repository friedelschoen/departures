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

SELECT DISTINCT ON (CASE WHEN e.route_short_name LIKE '^[0-9]+$'
         THEN CAST(e.route_short_name as integer)
         ELSE NULLIF(regexp_replace(e.route_short_name, '\D', '9', 'g'), '')::int
    END, e.route_id)
	e.route_short_name,
    e.route_color,
    e.route_text_color,
    e.route_id
FROM active_gtfs_stops s
JOIN active_gtfs_stop_events e 
    ON e.stop_id = s.stop_id 
    AND e.scheduled_time >= now() - interval '7 days'
WHERE s.stop_id = $1
    OR s.parent_station = $1
ORDER BY
    CASE WHEN e.route_short_name LIKE '^[0-9]+$'
         THEN CAST(e.route_short_name as integer)
         ELSE NULLIF(regexp_replace(e.route_short_name, '\D', '9', 'g'), '')::int
         END,
    e.route_id;
