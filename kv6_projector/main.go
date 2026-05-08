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
	"os"
	"time"

	_ "github.com/lib/pq"
)

const projectorName = "kv6-projector"

type Event struct {
	ID                        int64
	OperatingDay              time.Time
	DataOwnerCode             string
	LinePlanningNumber        string
	JourneyNumber             int
	ReinforcementNumber       int
	JourneyKey                string
	Status                    string
	EventTimestamp            time.Time
	Source                    string
	UserStopCode              sql.NullString
	PassageSequenceNumber     sql.NullInt64
	VehicleNumber             sql.NullInt64
	BlockCode                 sql.NullInt64
	Punctuality               sql.NullInt64
	RdX                       sql.NullInt64
	RdY                       sql.NullInt64
	DistanceSinceLastUserStop sql.NullInt64
}

func ensureoffset(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT INTO kv6_projection_offsets (
			projector_name,
			last_event_id
		)
		VALUES ($1, 0)
		ON CONFLICT (projector_name) DO NOTHING
	`, projectorName)

	return err
}

func readoffset(tx *sql.Tx) (int64, error) {
	var id int64

	err := tx.QueryRow(`
		SELECT last_event_id
		FROM kv6_projection_offsets
		WHERE projector_name = $1
	`, projectorName).Scan(&id)

	return id, err
}

func updateoffset(tx *sql.Tx, id int64) error {
	_, err := tx.Exec(`
		UPDATE kv6_projection_offsets
		SET last_event_id = $2,
		    updated_at = now()
		WHERE projector_name = $1
	`, projectorName, id)

	return err
}

func readevents(tx *sql.Tx, after int64, limit int) ([]Event, error) {
	rows, err := tx.Query(`
		SELECT
			id,
			operating_day,
			data_owner_code,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			realtime_trip_id,
			status,
			event_timestamp,
			source,
			user_stop_code,
			passage_sequence_number,
			vehicle_number,
			block_code,
			punctuality,
			rd_x,
			rd_y,
			distance_since_last_user_stop
		FROM kv6_events
		WHERE id > $1
		ORDER BY id
		LIMIT $2
	`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event

	for rows.Next() {
		var ev Event

		if err := rows.Scan(
			&ev.ID,
			&ev.OperatingDay,
			&ev.DataOwnerCode,
			&ev.LinePlanningNumber,
			&ev.JourneyNumber,
			&ev.ReinforcementNumber,
			&ev.JourneyKey,
			&ev.Status,
			&ev.EventTimestamp,
			&ev.Source,
			&ev.UserStopCode,
			&ev.PassageSequenceNumber,
			&ev.VehicleNumber,
			&ev.BlockCode,
			&ev.Punctuality,
			&ev.RdX,
			&ev.RdY,
			&ev.DistanceSinceLastUserStop,
		); err != nil {
			return nil, err
		}

		out = append(out, ev)
	}

	return out, rows.Err()
}

func projectcurrenttrip(tx *sql.Tx, ev Event) error {
	_, err := tx.Exec(`
		INSERT INTO kv6_current_trip (
			operating_day,
			data_owner_code,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			realtime_trip_id,
			status,
			event_timestamp,
			vehicle_number,
			block_code,
			user_stop_code,
			passage_sequence_number,
			punctuality,
			rd_x,
			rd_y,
			last_event_id
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16
		)
		ON CONFLICT (
			operating_day,
			data_owner_code,
			line_planning_number,
			trip_short_name,
			reinforcement_number
		)
		DO UPDATE SET
			realtime_trip_id = EXCLUDED.realtime_trip_id,
			status = EXCLUDED.status,
			event_timestamp = EXCLUDED.event_timestamp,
			vehicle_number = EXCLUDED.vehicle_number,
			block_code = EXCLUDED.block_code,
			user_stop_code = EXCLUDED.user_stop_code,
			passage_sequence_number = EXCLUDED.passage_sequence_number,
			punctuality = EXCLUDED.punctuality,
			rd_x = EXCLUDED.rd_x,
			rd_y = EXCLUDED.rd_y,
			last_event_id = EXCLUDED.last_event_id
		WHERE EXCLUDED.event_timestamp >= kv6_current_trip.event_timestamp
	`,
		ev.OperatingDay,
		ev.DataOwnerCode,
		ev.LinePlanningNumber,
		ev.JourneyNumber,
		ev.ReinforcementNumber,
		ev.JourneyKey,
		ev.Status,
		ev.EventTimestamp,
		nullint(ev.VehicleNumber),
		nullint(ev.BlockCode),
		nullstr(ev.UserStopCode),
		nullint(ev.PassageSequenceNumber),
		nullint(ev.Punctuality),
		nullint(ev.RdX),
		nullint(ev.RdY),
		ev.ID,
	)

	return err
}

func projectcurrentvehicle(tx *sql.Tx, ev Event) error {
	if !ev.VehicleNumber.Valid {
		return nil
	}

	_, err := tx.Exec(`
		INSERT INTO kv6_current_vehicle (
			operating_day,
			data_owner_code,
			vehicle_number,
			realtime_trip_id,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			status,
			event_timestamp,
			user_stop_code,
			passage_sequence_number,
			block_code,
			punctuality,
			rd_x,
			rd_y,
			last_event_id
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16
		)
		ON CONFLICT (
			operating_day,
			data_owner_code,
			vehicle_number
		)
		DO UPDATE SET
			realtime_trip_id = EXCLUDED.realtime_trip_id,
			line_planning_number = EXCLUDED.line_planning_number,
			trip_short_name = EXCLUDED.trip_short_name,
			reinforcement_number = EXCLUDED.reinforcement_number,
			status = EXCLUDED.status,
			event_timestamp = EXCLUDED.event_timestamp,
			user_stop_code = EXCLUDED.user_stop_code,
			passage_sequence_number = EXCLUDED.passage_sequence_number,
			block_code = EXCLUDED.block_code,
			punctuality = EXCLUDED.punctuality,
			rd_x = EXCLUDED.rd_x,
			rd_y = EXCLUDED.rd_y,
			last_event_id = EXCLUDED.last_event_id
		WHERE EXCLUDED.event_timestamp >= kv6_current_vehicle.event_timestamp
	`,
		ev.OperatingDay,
		ev.DataOwnerCode,
		ev.VehicleNumber.Int64,
		ev.JourneyKey,
		ev.LinePlanningNumber,
		ev.JourneyNumber,
		ev.ReinforcementNumber,
		ev.Status,
		ev.EventTimestamp,
		nullstr(ev.UserStopCode),
		nullint(ev.PassageSequenceNumber),
		nullint(ev.BlockCode),
		nullint(ev.Punctuality),
		nullint(ev.RdX),
		nullint(ev.RdY),
		ev.ID,
	)

	return err
}

func projecttripstopstatus(tx *sql.Tx, ev Event) error {
	if !ev.UserStopCode.Valid || !ev.PassageSequenceNumber.Valid {
		return nil
	}

	_, err := tx.Exec(`
		INSERT INTO kv6_trip_stop_status (
			operating_day,
			data_owner_code,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			realtime_trip_id,
			user_stop_code,
			passage_sequence_number,
			status,
			event_timestamp,
			vehicle_number,
			block_code,
			punctuality,
			rd_x,
			rd_y,
			last_event_id
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16
		)
		ON CONFLICT (
			operating_day,
			data_owner_code,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			user_stop_code,
			passage_sequence_number
		)
		DO UPDATE SET
			realtime_trip_id = EXCLUDED.realtime_trip_id,
			status = EXCLUDED.status,
			event_timestamp = EXCLUDED.event_timestamp,
			vehicle_number = EXCLUDED.vehicle_number,
			block_code = EXCLUDED.block_code,
			punctuality = EXCLUDED.punctuality,
			rd_x = EXCLUDED.rd_x,
			rd_y = EXCLUDED.rd_y,
			last_event_id = EXCLUDED.last_event_id
		WHERE EXCLUDED.event_timestamp >= kv6_trip_stop_status.event_timestamp
	`,
		ev.OperatingDay,
		ev.DataOwnerCode,
		ev.LinePlanningNumber,
		ev.JourneyNumber,
		ev.ReinforcementNumber,
		ev.JourneyKey,
		ev.UserStopCode.String,
		ev.PassageSequenceNumber.Int64,
		ev.Status,
		ev.EventTimestamp,
		nullint(ev.VehicleNumber),
		nullint(ev.BlockCode),
		nullint(ev.Punctuality),
		nullint(ev.RdX),
		nullint(ev.RdY),
		ev.ID,
	)

	return err
}

func projectvehiclehistory(tx *sql.Tx, ev Event) error {
	if !ev.VehicleNumber.Valid {
		return nil
	}

	_, err := tx.Exec(`
		INSERT INTO kv6_vehicle_trip_history (
			operating_day,
			data_owner_code,
			vehicle_number,
			realtime_trip_id,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			block_code,
			first_seen,
			last_seen,
			first_event_id,
			last_event_id
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12
		)
		ON CONFLICT (
			operating_day,
			data_owner_code,
			vehicle_number,
			realtime_trip_id
		)
		DO UPDATE SET
			block_code = COALESCE(EXCLUDED.block_code, kv6_vehicle_trip_history.block_code),
			first_seen = LEAST(kv6_vehicle_trip_history.first_seen, EXCLUDED.first_seen),
			last_seen = GREATEST(kv6_vehicle_trip_history.last_seen, EXCLUDED.last_seen),
			first_event_id = LEAST(kv6_vehicle_trip_history.first_event_id, EXCLUDED.first_event_id),
			last_event_id = GREATEST(kv6_vehicle_trip_history.last_event_id, EXCLUDED.last_event_id)
	`,
		ev.OperatingDay,
		ev.DataOwnerCode,
		ev.VehicleNumber.Int64,
		ev.JourneyKey,
		ev.LinePlanningNumber,
		ev.JourneyNumber,
		ev.ReinforcementNumber,
		nullint(ev.BlockCode),
		ev.EventTimestamp,
		ev.EventTimestamp,
		ev.ID,
		ev.ID,
	)

	return err
}

func projectblockhistory(tx *sql.Tx, ev Event) error {
	if !ev.BlockCode.Valid {
		return nil
	}

	_, err := tx.Exec(`
		INSERT INTO kv6_block_trip_history (
			operating_day,
			data_owner_code,
			block_code,
			realtime_trip_id,
			line_planning_number,
			trip_short_name,
			reinforcement_number,
			first_seen,
			last_seen,
			first_event_id,
			last_event_id
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11
		)
		ON CONFLICT (
			operating_day,
			data_owner_code,
			block_code,
			realtime_trip_id
		)
		DO UPDATE SET
			first_seen = LEAST(kv6_block_trip_history.first_seen, EXCLUDED.first_seen),
			last_seen = GREATEST(kv6_block_trip_history.last_seen, EXCLUDED.last_seen),
			first_event_id = LEAST(kv6_block_trip_history.first_event_id, EXCLUDED.first_event_id),
			last_event_id = GREATEST(kv6_block_trip_history.last_event_id, EXCLUDED.last_event_id)
	`,
		ev.OperatingDay,
		ev.DataOwnerCode,
		ev.BlockCode.Int64,
		ev.JourneyKey,
		ev.LinePlanningNumber,
		ev.JourneyNumber,
		ev.ReinforcementNumber,
		ev.EventTimestamp,
		ev.EventTimestamp,
		ev.ID,
		ev.ID,
	)

	return err
}

func nullint(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func nullstr(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func project(tx *sql.Tx, ev Event) error {
	if err := projectcurrenttrip(tx, ev); err != nil {
		return err
	}
	if err := projectcurrentvehicle(tx, ev); err != nil {
		return err
	}
	if err := projecttripstopstatus(tx, ev); err != nil {
		return err
	}
	if err := projectvehiclehistory(tx, ev); err != nil {
		return err
	}
	if err := projectblockhistory(tx, ev); err != nil {
		return err
	}

	return nil
}

func runbatch(db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if err := ensureoffset(tx); err != nil {
		return 0, err
	}

	last, err := readoffset(tx)
	if err != nil {
		return 0, err
	}

	events, err := readevents(tx, last, 1000)
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return 0, nil
	}

	for _, ev := range events {
		if err := project(tx, ev); err != nil {
			return 0, err
		}
		last = ev.ID
	}

	if err := updateoffset(tx, last); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(events), nil
}

func main() {
	db, err := sql.Open("postgres", os.Getenv("POSTGRES"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	for {
		now := time.Now()
		n, err := runbatch(db)
		if err != nil {
			log.Printf("project batch: %v", err)
		} else {
			log.Printf("projected %d kv6 events", n)
		}

		d := 5*time.Second - time.Since(now)
		time.Sleep(max(d, 0))
		log.Printf("waiting %v...", d)
		now = time.Now()
	}
}
