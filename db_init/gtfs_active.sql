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

CREATE OR REPLACE VIEW active_gtfs_stops AS
SELECT s.*
FROM gtfs_stops s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_routes AS
SELECT s.*
FROM gtfs_routes s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_trips AS
SELECT s.*
FROM gtfs_trips s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_trips AS
SELECT s.*
FROM gtfs_trips s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_calendar_dates AS
SELECT s.*
FROM gtfs_calendar_dates s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_trip_bounds AS
SELECT s.*
FROM gtfs_trip_bounds s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_shapes AS
SELECT s.*
FROM gtfs_shapes s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_agency AS
SELECT s.*
FROM gtfs_agency s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_stop_times AS
SELECT s.*
FROM gtfs_stop_times s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

CREATE OR REPLACE VIEW active_gtfs_stop_events AS
SELECT s.*
FROM gtfs_stop_events s
JOIN gtfs_feeds f ON f.id = s.feed_ref
WHERE f.active = true;

