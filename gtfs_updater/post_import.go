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

package main

import (
	"database/sql"
	"log"
)

var postImporters = []func(tx *sql.Tx, feedRef int64) error{
	generateMissingShapes,
	generateRealtimeSequence,
	calculateTripBounds,
}

func calculateTripBounds(tx *sql.Tx, feedRef int64) error {
	if _, err := tx.Exec(`SET LOCAL work_mem = '1GB'`); err != nil {
		return err
	}

	log.Println("Clearing trip_bounds...")
	if _, err := tx.Exec(`
	DELETE FROM gtfs_trip_bounds
	WHERE feed_ref = $1;
	`, feedRef); err != nil {
		return err
	}

	log.Println("Filling trip_bounds...")
	if _, err := tx.Exec(`
WITH bounds AS (
	SELECT
		feed_ref,
		trip_id,
		min(stop_sequence) AS start_sequence,
		max(stop_sequence) AS end_sequence
	FROM gtfs_stop_times
	WHERE feed_ref = $1
	GROUP BY feed_ref, trip_id
)
INSERT INTO gtfs_trip_bounds (
	feed_ref,
	trip_id,
	start_time,
	end_time,
	start_sequence,
	end_sequence,
	start_stop,
	end_stop
)
SELECT
	b.feed_ref,
	b.trip_id,
	COALESCE(start_st.departure_time, start_st.arrival_time) AS start_time,
	COALESCE(end_st.arrival_time, end_st.departure_time) AS end_time,
	b.start_sequence,
	b.end_sequence,
	start_st.stop_id AS start_stop,
	end_st.stop_id AS end_stop
FROM bounds b
JOIN gtfs_stop_times start_st
	ON start_st.feed_ref = b.feed_ref
   AND start_st.trip_id = b.trip_id
   AND start_st.stop_sequence = b.start_sequence
JOIN gtfs_stop_times end_st
	ON end_st.feed_ref = b.feed_ref
   AND end_st.trip_id = b.trip_id
   AND end_st.stop_sequence = b.end_sequence;`, feedRef); err != nil {
		return err
	}

	return nil
}

func generateRealtimeSequence(tx *sql.Tx, feedRef int64) error {
	log.Println("Generate realtime_trip_sequence...")
	if _, err := tx.Exec(`
WITH seq AS (
	SELECT
		feed_ref,
		trip_id,
		row_number() OVER (
			PARTITION BY feed_ref, service_id, realtime_trip_id
			ORDER BY trip_id
		) AS n
	FROM gtfs_trips
	WHERE feed_ref = $1
	  AND realtime_trip_id IS NOT NULL
)
UPDATE gtfs_trips t
SET realtime_trip_sequence = seq.n
FROM seq
WHERE t.feed_ref = seq.feed_ref
  AND t.trip_id = seq.trip_id;
 `, feedRef); err != nil {
		return err
	}

	return nil
}

func generateMissingShapes(tx *sql.Tx, feedRef int64) error {
	log.Println("Generate missing trip shapes...")
	_, err := tx.Exec(`
WITH missing AS (
	SELECT
		t.feed_ref,
		t.trip_id,
		'gtrip:' || t.trip_id AS shape_id
	FROM gtfs_trips t
	WHERE t.feed_ref = $1
	  AND NULLIF(t.shape_id, '') IS NULL
),
inserted_shapes AS (
	INSERT INTO gtfs_shapes (
		feed_ref,
		shape_id,
		shape_pt_sequence,
		shape_pt_lat,
		shape_pt_lon,
		shape_dist_traveled
	)
	SELECT
		m.feed_ref,
		m.shape_id,
		st.stop_sequence,
		s.stop_lat,
		s.stop_lon,
		NULL
	FROM missing m
	JOIN gtfs_stop_times st
		ON st.feed_ref = m.feed_ref
	   AND st.trip_id = m.trip_id
	JOIN gtfs_stops s
		ON s.feed_ref = st.feed_ref
	   AND s.stop_id = st.stop_id
	WHERE s.stop_lat IS NOT NULL
	  AND s.stop_lon IS NOT NULL
	ORDER BY
		m.feed_ref,
		m.trip_id,
		st.stop_sequence
	ON CONFLICT DO NOTHING
)
UPDATE gtfs_trips t
SET shape_id = m.shape_id
FROM missing m
WHERE t.feed_ref = m.feed_ref
  AND t.trip_id = m.trip_id
`, feedRef)

	return err
}

func runPostImporters(db *sql.DB, feedRef int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, fn := range postImporters {
		if err := fn(tx, feedRef); err != nil {
			return err
		}
	}

	return tx.Commit()
}
