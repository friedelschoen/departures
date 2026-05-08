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
	sh.shape_pt_lat,
	sh.shape_pt_lon,
	sh.shape_pt_sequence
FROM active_gtfs_trips t
JOIN active_gtfs_shapes sh
    ON sh.shape_id = t.shape_id
WHERE t.trip_id = $1
ORDER BY sh.shape_pt_sequence;
