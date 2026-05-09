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
	"strconv"
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
		ON CONFLICT (feed_id, feed_version)
		DO UPDATE SET imported_at = now()
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

func iterCSV(tmpdir string, filename string, fn func(map[string]string) error) error {
	log.Println("open ", filename)
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

	var total int

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
		total++

		off := r.InputOffset()
		if off-prevOff > 1000000 {
			log.Printf("%s - %5.1f%% - %dMB / %dMB\n", filename, 100*float64(off)/float64(totalSize), off/1000000, totalSize/1000000)

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

		if err := fn(row); err != nil {
			return err
		}
	}
	log.Printf("done %s - %dMB\n", filename, totalSize/1000000)
	return err
}

func insertCSV(stmt *sql.Stmt, tmpdir string, filename string, fn func(map[string]string) []any, cleanup func() [][]any) error {
	var total, skipped int

	err := iterCSV(tmpdir, filename, func(row map[string]string) error {
		total++
		insert := fn(row)
		if len(insert) == 0 {
			skipped++
			return nil
		}
		_, err := stmt.Exec(insert...)
		return err
	})
	if cleanup != nil {
		for _, insert := range cleanup() {
			if _, err := stmt.Exec(insert...); err != nil {
				return err
			}
		}
	}
	_, err = stmt.Exec()
	log.Printf("%d of %d skipped (%.1f)\n", skipped, total, 100*float64(skipped)/float64(total))
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
	if s == "" {
		return nil
	}
	return s
}

func unpackFile(dest string, f *zip.File) error {
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
	log.Printf("unpacked %s - %d bytes copied\n", f.Name, n)
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

func importTables(tx *sql.Tx, feedRef int64, zr string) error {
	log.Printf("collecting...\n")

	routes, agencies, err := collectRoutes(zr)
	if err != nil {
		return err
	}

	trips, services, shapes, err := collectTrips(zr, routes)
	if err != nil {
		return err
	}

	stops, err := collectStopTimes(zr, trips)
	if err != nil {
		return err
	}

	err = collectStops(zr, stops)
	if err != nil {
		return err
	}

	log.Printf("importing new feed_ref=%d...\n", feedRef)

	err = importAgency(tx, feedRef, zr, agencies)
	if err != nil {
		return err
	}
	err = importRoutes(tx, feedRef, zr)
	if err != nil {
		return err
	}
	err = importTrips(tx, feedRef, zr, routes)
	if err != nil {
		return err
	}
	err = importCalendarDates(tx, feedRef, zr, services)
	if err != nil {
		return err
	}
	err = importStops(tx, feedRef, zr, stops)
	if err != nil {
		return err
	}
	err = importStopTimes(tx, feedRef, zr, trips)
	if err != nil {
		return err
	}
	err = importShapes(tx, feedRef, zr, shapes)
	if err != nil {
		return err
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

	if err := importTables(tx, feedRef, zr); err != nil {
		return 0, err
	}

	log.Println("Import done! Finishing...")

	// insert done!

	err = runPostImporters(tx, feedRef)
	if err != nil {
		return 0, err
	}

	if err := activateFeed(tx, feedRef); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
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
