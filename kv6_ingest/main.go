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
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	zmq "github.com/pebbe/zmq4"
)

var topics = [...]string{
	"/QBUZZ/KV6posinfo",
}

type kv6Subscription struct {
	XMLName xml.Name     `xml:"VV_TM_PUSH"`
	PosInfo []kv6PosInfo `xml:"KV6posinfo"`
}

type kv6PosInfo struct {
	XMLName xml.Name  `xml:"KV6posinfo"`
	Items   []kv6Item `xml:",any"`
}

type kv6Item struct {
	XMLName xml.Name

	OperatingDay              string `xml:"operatingday"`
	DataOwnerCode             string `xml:"dataownercode"`
	LinePlanningNumber        string `xml:"lineplanningnumber"`
	JourneyNumber             int    `xml:"journeynumber"`
	ReinforcementNumber       int    `xml:"reinforcementnumber"`
	Timestamp                 string `xml:"timestamp"`
	Source                    string `xml:"source"`
	UserStopCode              string `xml:"userstopcode"`
	PassageSequenceNumber     int    `xml:"passagesequencenumber"`
	VehicleNumber             int    `xml:"vehiclenumber"`
	BlockCode                 int    `xml:"blockcode"`
	WheelchairAccessible      string `xml:"wheelchairaccessible"`
	NumberOfCoaches           int    `xml:"numberofcoaches"`
	Punctuality               int    `xml:"punctuality"`
	RdX                       int    `xml:"rd-x"`
	RdY                       int    `xml:"rd-y"`
	DistanceSinceLastUserStop int    `xml:"distancesincelastuserstop"`
}

func journeykey(item kv6Item) string {
	if item.ReinforcementNumber > 0 {
		return fmt.Sprintf(
			"%s:%s:%d:%d",
			item.DataOwnerCode,
			item.LinePlanningNumber,
			item.JourneyNumber,
			item.ReinforcementNumber,
		)
	}

	return fmt.Sprintf(
		"%s:%s:%d",
		item.DataOwnerCode,
		item.LinePlanningNumber,
		item.JourneyNumber,
	)
}

func nullint(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullstr(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func insertEvent(db *sql.DB, item kv6Item) error {
	status := item.XMLName.Local

	operatingDay, err := time.Parse(time.DateOnly, item.OperatingDay)
	if err != nil {
		return fmt.Errorf("invalid operating day %q: %w", item.OperatingDay, err)
	}

	eventTimestamp, err := time.Parse(time.RFC3339, item.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp %q: %w", item.Timestamp, err)
	}

	var (
		userStopCode              any
		passageSequenceNumber     any
		vehicleNumber             any
		blockCode                 any
		punctuality               any
		rdX                       any
		rdY                       any
		distanceSinceLastUserStop any
	)

	switch status {
	case "DELAY":
		punctuality = item.Punctuality

	case "INIT":
		userStopCode = nullstr(item.UserStopCode)
		passageSequenceNumber = nullint(item.PassageSequenceNumber)
		vehicleNumber = nullint(item.VehicleNumber)
		blockCode = nullint(item.BlockCode)

	case "ARRIVAL", "ONSTOP", "DEPARTURE", "ONROUTE":
		userStopCode = nullstr(item.UserStopCode)
		passageSequenceNumber = nullint(item.PassageSequenceNumber)
		vehicleNumber = nullint(item.VehicleNumber)
		punctuality = item.Punctuality

		if item.RdX > 0 && item.RdY > 0 {
			rdX = item.RdX
			rdY = item.RdY
		}

		if status == "ONROUTE" {
			distanceSinceLastUserStop = nullint(item.DistanceSinceLastUserStop)
		}

	case "OFFROUTE", "END":
		userStopCode = nullstr(item.UserStopCode)
		passageSequenceNumber = nullint(item.PassageSequenceNumber)
		vehicleNumber = nullint(item.VehicleNumber)

		if item.RdX > 0 && item.RdY > 0 {
			rdX = item.RdX
			rdY = item.RdY
		}
	}

	_, err = db.Exec(`
		INSERT INTO kv6_events (
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
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17
		)
	`,
		operatingDay,
		item.DataOwnerCode,
		item.LinePlanningNumber,
		item.JourneyNumber,
		item.ReinforcementNumber,
		journeykey(item),
		status,
		eventTimestamp,
		item.Source,
		userStopCode,
		passageSequenceNumber,
		vehicleNumber,
		blockCode,
		punctuality,
		rdX,
		rdY,
		distanceSinceLastUserStop,
	)

	return err
}

func handleKV6(db *sql.DB, r io.Reader) error {
	var sub kv6Subscription

	dec := xml.NewDecoder(r)
	if err := dec.Decode(&sub); err != nil {
		return err
	}

	for _, posinfo := range sub.PosInfo {
		for _, item := range posinfo.Items {
			if err := insertEvent(db, item); err != nil {
				log.Printf("insert kv6 event: %v", err)
			}
		}
	}

	return nil
}

func handleMessage(db *sql.DB, topic string, buf []byte) error {
	switch topic {
	case "/QBUZZ/KV6posinfo":
		r, err := gzip.NewReader(bytes.NewBuffer(buf))
		if err != nil {
			return err
		}
		defer r.Close()

		return handleKV6(db, r)
	}

	return nil
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

	ctx, err := zmq.NewContext()
	if err != nil {
		log.Fatal(err)
	}
	defer ctx.Term()

	sub, err := ctx.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Close()

	for _, topic := range topics {
		if err := sub.SetSubscribe(topic); err != nil {
			log.Fatalf("subscribe %s: %v", topic, err)
		}
	}

	endpoint := os.Getenv("BISON_MQ")
	if err := sub.Connect(endpoint); err != nil {
		log.Fatal(err)
	}

	log.Printf("connected to %s", endpoint)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	poller := zmq.NewPoller()
	poller.Add(sub, zmq.POLLIN)

	for {
		select {
		case <-interrupt:
			log.Println("stopping")
			return
		default:
		}

		sockets, err := poller.Poll(time.Second)
		if err != nil {
			log.Println("poll:", err)
			continue
		}

		for _, socket := range sockets {
			if socket.Socket != sub {
				continue
			}

			parts, err := sub.RecvMessageBytes(0)
			if err != nil {
				log.Println("recv:", err)
				continue
			}
			if len(parts) != 2 {
				continue
			}

			if err := handleMessage(db, string(parts[0]), parts[1]); err != nil {
				log.Println("handle:", err)
			}
		}
	}
}
