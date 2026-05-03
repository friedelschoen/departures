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
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const staleAfter = 12 * time.Hour

func activeFeedIsStale(db *sql.DB) (bool, error) {
	var importedAt time.Time

	err := db.QueryRow(`
		SELECT imported_at
		FROM gtfs_feeds
		WHERE active = true
		ORDER BY imported_at DESC
		LIMIT 1
	`).Scan(&importedAt)

	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return time.Since(importedAt) >= staleAfter, nil
}

func download(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "DepartureBot/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func insertFeed(tx *sql.Tx, row map[string]string) (int64, bool, error) {
	var id int64
	var inserted bool

	err := tx.QueryRow(`
		INSERT INTO gtfs_feeds (
			feed_publisher_name,
			feed_id,
			feed_publisher_url,
			feed_lang,
			feed_start_date,
			feed_end_date,
			feed_version,
			active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false)
--		ON CONFLICT (feed_id, feed_version)
--		DO UPDATE SET imported_at = now()
		RETURNING id, xmax = 0
	`,
		nullString(row["feed_publisher_name"]),
		nullString(row["feed_id"]),
		nullString(row["feed_publisher_url"]),
		nullString(row["feed_lang"]),
		parseGTFSDate(row["feed_start_date"]),
		parseGTFSDate(row["feed_end_date"]),
		nullString(row["feed_version"]),
	).Scan(&id, &inserted)

	return id, inserted, err
}

func activateFeed(tx *sql.Tx, feedRef int64) error {
	if _, err := tx.Exec(`UPDATE gtfs_feeds SET active = false`); err != nil {
		return err
	}

	_, err := tx.Exec(`
		UPDATE gtfs_feeds
		SET active = true, imported_at = now()
		WHERE id = $1
	`, feedRef)

	return err
}

func importAgency(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_agency (
				feed_ref,
				agency_id,
				agency_name,
				agency_url,
				agency_timezone,
				agency_phone
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "agency.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["agency_id"],
			row["agency_name"],
			nullString(row["agency_url"]),
			nullString(row["agency_timezone"]),
			nullString(row["agency_phone"]),
		}
	})
}

func importCalendarDates(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_calendar_dates (
				feed_ref,
				service_id,
				date,
				exception_type
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "calendar_dates.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["service_id"],
			parseGTFSDate(row["date"]),
			parseInt(row["exception_type"]),
		}
	})
}

func importRoutes(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_routes (
				feed_ref,
				route_id,
				agency_id,
				route_short_name,
				route_long_name,
				route_desc,
				route_type,
				route_color,
				route_text_color,
				route_url
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "routes.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["route_id"],
			nullString(row["agency_id"]),
			nullString(row["route_short_name"]),
			nullString(row["route_long_name"]),
			nullString(row["route_desc"]),
			parseNullableInt(row["route_type"]),
			nullString(row["route_color"]),
			nullString(row["route_text_color"]),
			nullString(row["route_url"]),
		}
	})
}

func importShapes(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_shapes (
				feed_ref,
				shape_id,
				shape_pt_sequence,
				shape_pt_lat,
				shape_pt_lon,
				shape_dist_traveled
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "shapes.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["shape_id"],
			parseInt(row["shape_pt_sequence"]),
			parseFloat(row["shape_pt_lat"]),
			parseFloat(row["shape_pt_lon"]),
			parseNullableFloat(row["shape_dist_traveled"]),
		}
	})
}

func importStops(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_stops (
				feed_ref,
				stop_id,
				stop_code,
				stop_name,
				stop_lat,
				stop_lon,
				location_type,
				parent_station,
				stop_timezone,
				wheelchair_boarding,
				platform_code,
				zone_id
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "stops.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["stop_id"],
			nullString(row["stop_code"]),
			nullString(row["stop_name"]),
			parseNullableFloat(row["stop_lat"]),
			parseNullableFloat(row["stop_lon"]),
			parseNullableInt(row["location_type"]),
			nullString(row["parent_station"]),
			nullString(row["stop_timezone"]),
			parseNullableInt(row["wheelchair_boarding"]),
			nullString(row["platform_code"]),
			nullString(row["zone_id"]),
		}
	})
}

func importTrips(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_trips (
				feed_ref,
				route_id,
				service_id,
				trip_id,
				realtime_trip_id,
				trip_headsign,
				trip_short_name,
				trip_long_name,
				direction_id,
				block_id,
				shape_id,
				wheelchair_accessible,
				bikes_allowed
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "trips.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["route_id"],
			row["service_id"],
			row["trip_id"],
			nullString(row["realtime_trip_id"]),
			nullString(row["trip_headsign"]),
			nullString(row["trip_short_name"]),
			nullString(row["trip_long_name"]),
			parseNullableInt(row["direction_id"]),
			nullString(row["block_id"]),
			nullString(row["shape_id"]),
			parseNullableInt(row["wheelchair_accessible"]),
			parseNullableInt(row["bikes_allowed"]),
		}
	})
}

func importStopTimes(tx *sql.Tx, feedRef int64, a string) error {
	stmt, err := tx.Prepare(`
			COPY gtfs_stop_times (
				feed_ref,
				trip_id,
				stop_sequence,
				stop_id,
				stop_headsign,
				arrival_time,
				departure_time,
				pickup_type,
				drop_off_type,
				timepoint,
				shape_dist_traveled,
				fare_units_traveled
			) FROM STDIN
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return insertCSV(stmt, a, "stop_times.txt", func(row map[string]string) []any {
		return []any{
			feedRef,
			row["trip_id"],
			parseInt(row["stop_sequence"]),
			row["stop_id"],
			nullString(row["stop_headsign"]),
			nullString(row["arrival_time"]),
			nullString(row["departure_time"]),
			parseNullableInt(row["pickup_type"]),
			parseNullableInt(row["drop_off_type"]),
			parseNullableInt(row["timepoint"]),
			parseNullableFloat(row["shape_dist_traveled"]),
			parseNullableInt(row["fare_units_traveled"]),
		}
	})
}

func readFirstRow(rc io.Reader) (map[string]string, error) {
	r := csv.NewReader(rc)
	r.FieldsPerRecord = -1

	keys, err := r.Read()
	if err != nil {
		return nil, err
	}
	values, err := r.Read()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if i < len(values) {
			result[key] = values[i]
		} else {
			result[key] = ""
		}
	}
	return result, nil
}

func insertCSV(stmt *sql.Stmt, tmpdir string, filename string, fn func(map[string]string) []any) error {
	fmt.Println("open ", filename)
	rc, err := os.Open(filepath.Join(tmpdir, filename))
	if err != nil {
		return err
	}
	defer rc.Close()

	var totalSize int64
	if fileinfo, err := rc.Stat(); err == nil {
		totalSize = fileinfo.Size()
	}

	r := csv.NewReader(rc)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return err
	}

	var prevOff int64
	row := make(map[string]string, len(header))
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		off := r.InputOffset()
		if off-prevOff > 1000000 {
			fmt.Printf("%s - %5.1f%% - %dMB / %dMB\n", filename, 100*float64(off)/float64(totalSize), off/1000000, totalSize/1000000)

			prevOff = off
		}

		for k := range row {
			row[k] = ""
		}
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}

		insert := fn(row)
		if _, err := stmt.Exec(insert...); err != nil {
			return err
		}
	}
	_, err = stmt.Exec()
	fmt.Printf("%s - 100.0%% - %dMB / %dMB\n", filename, totalSize/1000000, totalSize/1000000)
	return err
}

func parseGTFSDate(s string) any {
	if s == "" {
		return nil
	}

	t, err := time.Parse("20060102", s)
	if err != nil {
		return nil
	}

	return t
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseNullableInt(s string) any {
	if s == "" {
		return nil
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}

	return i
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseNullableFloat(s string) any {
	if s == "" {
		return nil
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}

	return f
}

func nullString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func calculateTripBounds(tx *sql.DB, feedRef int64) error {
	if _, err := tx.Exec(`SET LOCAL work_mem = '1GB'`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
	DELETE FROM gtfs_trip_bounds
	WHERE feed_ref = $1;
	`, feedRef); err != nil {
		return err
	}

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
	end_sequence
)
SELECT
	b.feed_ref,
	b.trip_id,
	COALESCE(start_st.departure_time, start_st.arrival_time),
	COALESCE(end_st.arrival_time, end_st.departure_time),
	b.start_sequence,
	b.end_sequence
FROM bounds b
JOIN gtfs_stop_times start_st
	ON start_st.feed_ref = b.feed_ref
   AND start_st.trip_id = b.trip_id
   AND start_st.stop_sequence = b.start_sequence
JOIN gtfs_stop_times end_st
	ON end_st.feed_ref = b.feed_ref
   AND end_st.trip_id = b.trip_id
   AND end_st.stop_sequence = b.end_sequence;
`, feedRef); err != nil {
		return err
	}

	return nil
}

var gtfsFiles = []string{"agency.txt", "feed_info.txt", "shapes.txt", "stops.txt", "trips.txt",
	"calendar_dates.txt", "routes.txt", "stop_times.txt", "transfers.txt"}

func unpackFile(dest string, f *zip.File) error {
	if !slices.Contains(gtfsFiles, f.Name) {
		return nil
	}

	instream, err := f.Open()
	if err != nil {
		return err
	}
	defer instream.Close()

	destpath := filepath.Join(dest, f.Name)
	outstream, err := os.Create(destpath)
	if err != nil {
		return err
	}
	defer outstream.Close()

	n, err := io.Copy(outstream, instream)
	if err != nil {
		return err
	}
	fmt.Printf("unpacked %s - %d bytes copied\n", f.Name, n)
	return nil
}

func unpack(dest string, buf []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		if err := unpackFile(dest, f); err != nil {
			return err
		}
	}
	return nil
}

func importData(db *sql.DB, gtfsURL string) (int64, error) {
	needsUpdate, err := activeFeedIsStale(db)
	if err != nil {
		return 0, err
	}
	if !needsUpdate {
		return 0, os.ErrExist
	}

	log.Println("downloading GTFS:", gtfsURL)

	buf, err := download(gtfsURL)
	if err != nil {
		return 0, err
	}

	zr, err := os.MkdirTemp("", "gtfs_updater")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(zr)

	if err := unpack(zr, buf); err != nil {
		return 0, err
	}

	file, err := os.Open(filepath.Join(zr, "feed_info.txt"))
	if err != nil {
		return 0, err
	}
	feedInfo, err := readFirstRow(file)
	if err != nil {
		return 0, err
	}
	file.Close()

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	feedRef, imported, err := insertFeed(tx, feedInfo)
	if err != nil {
		return 0, err
	}

	if !imported {
		log.Printf("feed already exists; activating feed_ref=%d\n", feedRef)
		if err := activateFeed(tx, feedRef); err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return 0, os.ErrExist
	}

	log.Printf("importing new feed_ref=%d\n", feedRef)

	err = importAgency(tx, feedRef, zr)
	if err != nil {
		return 0, err
	}
	err = importCalendarDates(tx, feedRef, zr)
	if err != nil {
		return 0, err
	}
	err = importRoutes(tx, feedRef, zr)
	if err != nil {
		return 0, err
	}
	// err = importShapes(tx, feedRef, zr)
	// if err != nil {
	// 	return err
	// }
	err = importStops(tx, feedRef, zr)
	if err != nil {
		return 0, err
	}
	err = importTrips(tx, feedRef, zr)
	if err != nil {
		return 0, err
	}
	err = importStopTimes(tx, feedRef, zr)
	if err != nil {
		return 0, err
	}

	if err := activateFeed(tx, feedRef); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	// insert done!

	err = calculateTripBounds(db, feedRef)
	if err != nil {
		return 0, err
	}
	return feedRef, nil
}

func main() {
	dburl := os.Getenv("POSTGRES")
	if dburl == "" {
		log.Fatal("missing POSTGRES env")
	}

	gtfsURL := os.Getenv("GTFS_URL")
	if gtfsURL == "" {
		log.Fatal("missing GTFS_URL env")
	}

	db, err := sql.Open("postgres", dburl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	for {
		feedRef, err := importData(db, gtfsURL)
		if err == os.ErrExist {
			log.Printf("GTFS import already exist\n")
			time.Sleep(12 * time.Hour)
		} else if err != nil {
			log.Printf("GTFS import failed: %v\n", err)
			time.Sleep(10 * time.Minute)
		} else {
			log.Printf("GTFS import complete; active feed_ref=%d\n", feedRef)
			time.Sleep(12 * time.Hour)
		}
	}
}
