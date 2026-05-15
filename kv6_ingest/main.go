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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dylandreimerink/go-rijksdriehoek"
	_ "github.com/lib/pq"
	zmq "github.com/pebbe/zmq4"
)

const endpoint = "tcp://pubsub.besteffort.ndovloket.nl:7658"

var topics = [...]string{
	"/QBUZZ/KV6posinfo",
	//	"/QBUZZ/KV15messages",
	//  "/QBUZZ/KV17cvlinfo",
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

	OperatingDay              string `xml:"operatingday"` // date (2015-12-02)
	DataOwnerCode             string `xml:"dataownercode"`
	LinePlanningNumber        string `xml:"lineplanningnumber"`
	JourneyNumber             int    `xml:"journeynumber"`
	ReinforcementNumber       int    `xml:"reinforcementnumber"`
	Timestamp                 string `xml:"timestamp"` // ISO8601p5
	Source                    string `xml:"source"`    // VEHICLE,SERVER
	UserStopCode              string `xml:"userstopcode"`
	PassageSequenceNumber     int    `xml:"passagesequencenumber"`
	VehicleNumber             int    `xml:"vehiclenumber"`
	BlockCode                 int    `xml:"blockcode"`
	WheelchairAccessible      string `xml:"wheelchairaccessible"` // ACCESSIBLE,NOTACCESSIBLE,UNKNOWN -> true,false,null
	NumberOfCoaches           int    `xml:"numberofcoaches"`
	Punctuality               int    `xml:"punctuality"`
	RdX                       int    `xml:"rd-x"`
	RdY                       int    `xml:"rd-y"`
	DistanceSinceLastUserStop int    `xml:"distancesincelastuserstop"`
}

func placeholders(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("$%d", i+1)
	}
	return out
}

func handleKV6(dbpool *sql.DB, r io.Reader) error {
	var sub kv6Subscription

	dec := xml.NewDecoder(r)
	if err := dec.Decode(&sub); err != nil {
		return err
	}

	for _, posinfo := range sub.PosInfo {
		for _, item := range posinfo.Items {
			status := item.XMLName.Local
			operatingDay, err := time.Parse(time.DateOnly, item.OperatingDay)
			if err != nil {
				log.Printf("invalid date `%s`: %v\n", item.OperatingDay, err)
				continue
			}
			timestamp, err := time.Parse(time.RFC3339, item.Timestamp)
			if err != nil {
				log.Printf("invalid timestamp `%s`: %v\n", item.Timestamp, err)
				continue
			}
			realtimeTripID := []string{item.DataOwnerCode, item.LinePlanningNumber, strconv.Itoa(item.JourneyNumber)}
			if item.ReinforcementNumber > 0 {
				realtimeTripID = append(realtimeTripID, strconv.Itoa(item.JourneyNumber))
			}
			keys := []string{
				"operating_day",
				"data_owner_code",
				"line_planning_number",
				"trip_short_name",
				"reinforcement_number",
				"realtime_trip_id",

				"status",
				"source",
				"event_timestamp",
			}
			values := []any{
				operatingDay,
				item.DataOwnerCode,
				item.LinePlanningNumber,
				item.JourneyNumber,
				item.ReinforcementNumber,
				strings.Join(realtimeTripID, ":"),

				status,
				item.Source,
				timestamp,
			}

			switch status {
			case "DELAY":
				keys = append(keys, "punctuality")
				values = append(values, item.Punctuality)

			case "INIT":
				var wheelchair *bool
				switch item.WheelchairAccessible {
				case "ACCESSIBLE":
					wheelchair = new(true)
				case "NOTACCESSIBLE":
					wheelchair = new(false)
				}

				keys = append(keys,
					"user_stop_code",
					"passage_sequence_number",
					"vehicle_number",
					"block_code",
					"wheelchair_accessible",
					"number_of_coaches")

				values = append(values,
					item.UserStopCode,
					item.PassageSequenceNumber,
					item.VehicleNumber,
					item.BlockCode,
					wheelchair,
					item.NumberOfCoaches)

			case "ARRIVAL", "ONSTOP", "DEPARTURE", "ONROUTE":
				keys = append(keys,
					"user_stop_code",
					"passage_sequence_number",
					"vehicle_number",
					"punctuality")

				values = append(values,
					item.UserStopCode,
					item.PassageSequenceNumber,
					item.VehicleNumber,
					item.Punctuality)

				// In the Netherlands, rd-x and rd-y are always positive
				if item.RdX > 0 || item.RdY > 0 {
					lat, lon := rijksdriehoek.RDtoWGS84(float64(item.RdX), float64(item.RdY))
					keys = append(keys, "rd_x", "rd_y", "lat", "lon")
					values = append(values, item.RdX, item.RdY, lat, lon)
				}

				if status == "ONROUTE" {
					keys = append(keys, "distance_since_last_user_stop")
					values = append(values, item.DistanceSinceLastUserStop)
				}

			case "OFFROUTE", "END":
				keys = append(keys,
					"user_stop_code",
					"passage_sequence_number",
					"vehicle_number")

				values = append(values,
					item.UserStopCode,
					item.PassageSequenceNumber,
					item.VehicleNumber)

				// In the Netherlands, rd-x and rd-y are always positive
				if item.RdX > 0 || item.RdY > 0 {
					lat, lon := rijksdriehoek.RDtoWGS84(float64(item.RdX), float64(item.RdY))
					keys = append(keys, "rd_x", "rd_y", "lat", "lon")
					values = append(values, item.RdX, item.RdY, lat, lon)
				}
			}

			_, err = dbpool.Exec(fmt.Sprintf(
				"INSERT INTO kv6_events (%s) VALUES (%s);",
				strings.Join(keys, ", "),
				strings.Join(placeholders(len(values)), ", "),
			), values...)
			if err != nil {
				log.Printf("error in sql: %v\n", err)
				continue
			}
		}
	}
	return nil
}

func handleMessage(dbpool *sql.DB, topic string, buffer []byte) error {
	switch topic {
	case "/QBUZZ/KV6posinfo":
		unzipped, err := gzip.NewReader(bytes.NewBuffer(buffer))
		if err != nil {
			return err
		}
		defer unzipped.Close()

		if err := handleKV6(dbpool, unzipped); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	dbpool, err := sql.Open("postgres", os.Getenv("POSTGRES"))
	if err != nil {
		log.Fatal(err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(); err != nil {
		log.Fatalln(err)
		return
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
			return
		}
	}

	if err := sub.Connect(endpoint); err != nil {
		log.Fatal(err)
	}

	log.Printf("connected to %s", endpoint)
	for _, topic := range topics {
		log.Printf("subscribed to %s", topic)
	}

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

		sockets, err := poller.Poll(1 * time.Second)
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

			topic := string(parts[0])
			log.Printf("received %s\n", topic)
			if err := handleMessage(dbpool, topic, parts[1]); err != nil {
				log.Println("recv: ", err)
			}
		}
	}
}
