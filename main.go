package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/friedelschoen/depatures/gtfs"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=. --go_opt=module=github.com/friedelschoen/depatures -Iproto proto/gtfs-realtime-OVapi.proto proto/gtfs-realtime.proto

const (
	staticURL           = "https://gtfs.ovapi.nl/nl/gtfs-nl.zip"
	tripUpdatesURL      = "https://gtfs.ovapi.nl/nl/tripUpdates.pb"
	vehiclePositionsURL = "https://gtfs.ovapi.nl/nl/vehiclePositions.pb"

	defaultStopID = "3925913" /* change this */
)

type Static struct {
	Stops       map[string]Stop
	Routes      map[string]Route
	Trips       map[string]Trip
	StopTimes   map[string][]StopTime
	Calendar    map[string]Calendar
	DateChanges map[string]map[string]int
}

type Stop struct {
	ID   string
	Name string
}

type Route struct {
	ID        string
	ShortName string
	LongName  string
}

type Trip struct {
	ID        string
	RouteID   string
	ServiceID string
	Headsign  string
}

type StopTime struct {
	TripID    string
	StopID    string
	Departure int
	Headsign  string
}

type Calendar struct {
	Weekdays [7]bool
	Start    time.Time
	End      time.Time
}

type Realtime struct {
	TripUpdates map[string]*gtfs.TripUpdate
	Vehicles    map[string]*gtfs.VehiclePosition
	Updated     time.Time
}

type Server struct {
	static Static

	mu sync.RWMutex
	rt Realtime
}

type Departure struct {
	Line          string   `json:"line"`
	RouteID       string   `json:"route_id"`
	TripID        string   `json:"trip_id"`
	Headsign      string   `json:"headsign"`
	ScheduledTime string   `json:"scheduled_time"`
	RealtimeTime  string   `json:"realtime_time"`
	DelayMinutes  int      `json:"delay_minutes"`
	Cancelled     bool     `json:"cancelled"`
	Vehicle       *Vehicle `json:"vehicle,omitempty"`
}

type Vehicle struct {
	ID  string  `json:"id"`
	Lat float32 `json:"lat"`
	Lon float32 `json:"lon"`
}

func main() {
	cache := cacheFile("cache/static.zip")
	st, err := loadStatic(staticURL, &cache)
	if err != nil {
		log.Fatal(err)
	}
	cache.buffer = nil

	s := &Server{static: st}
	go s.refreshLoop()

	http.HandleFunc("/", s.index)
	http.HandleFunc("/api/departures", s.departures)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) refreshLoop() {
	tripUpdatesCache := cacheFile("cache/tripUpdates.pb")
	vehiclePositionsCache := cacheFile("cache/vehiclePositions.pb")

	for {
		rt := Realtime{
			TripUpdates: map[string]*gtfs.TripUpdate{},
			Vehicles:    map[string]*gtfs.VehiclePosition{},
			Updated:     time.Now(),
		}

		if feed, err := fetchFeed(tripUpdatesURL, &tripUpdatesCache); err == nil {
			for _, e := range feed.GetEntity() {
				tu := e.GetTripUpdate()
				if tu == nil || tu.GetTrip() == nil {
					continue
				}
				if id := tu.GetTrip().GetTripId(); id != "" {
					rt.TripUpdates[id] = tu
				}
			}
		} else {
			log.Println("tripUpdates:", err)
		}

		if feed, err := fetchFeed(vehiclePositionsURL, &vehiclePositionsCache); err == nil {
			for _, e := range feed.GetEntity() {
				v := e.GetVehicle()
				if v == nil || v.GetTrip() == nil {
					continue
				}
				if id := v.GetTrip().GetTripId(); id != "" {
					rt.Vehicles[id] = v
				}
			}
		} else {
			log.Println("vehiclePositions:", err)
		}

		s.mu.Lock()
		s.rt = rt
		s.mu.Unlock()

		time.Sleep(120 * time.Second)
	}
}

type cache struct {
	path   string
	buffer []byte
	time   time.Time
}

func cacheFile(path string) (c cache) {
	c.path = path

	stat, err := os.Stat(path)
	if err != nil {
		log.Printf("unable to stat cache %s: %v\n", path, err)
		return c
	}

	b, err := os.ReadFile(path)
	if err != nil {
		log.Printf("unable to read cache %s: %v\n", path, err)
		return c
	}

	c.time = stat.ModTime()
	c.buffer = b
	return
}

func (cache *cache) update(url string) error {
	fmt.Printf("updating %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DepatureBot/0.1 <derfriedmundschoen@gmail.com>")
	if !cache.time.IsZero() {
		req.Header.Set("If-Modified-Since", cache.time.Format(time.RFC1123))
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("%s: %s\n", url, resp.Status)
	switch resp.StatusCode {
	case http.StatusOK:
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		cache.buffer = b
		cache.time = time.Now()

		if err := os.WriteFile(cache.path, cache.buffer, 0644); err != nil {
			log.Printf("unable to write to cache %s: %v\n", cache.path, err)
		}

	case http.StatusNotModified:
		/* it's fine, we'll use cache! */

	default:
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	return nil
}

func fetchFeed(url string, cache *cache) (*gtfs.FeedMessage, error) {
	if err := cache.update(url); err != nil {
		return nil, err
	}
	var msg gtfs.FeedMessage
	if err := proto.Unmarshal(cache.buffer, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	stop := r.URL.Query().Get("stop")
	if stop == "" {
		stop = defaultStopID
	}

	page.Execute(w, map[string]string{
		"StopID": stop,
	})
}

func (s *Server) departures(w http.ResponseWriter, r *http.Request) {
	stopID := r.URL.Query().Get("stop")
	if stopID == "" {
		stopID = defaultStopID
	}

	now := time.Now()
	base := midnight(now)

	s.mu.RLock()
	rt := s.rt
	s.mu.RUnlock()

	var out []Departure

	for _, st := range s.static.StopTimes[stopID] {
		trip, ok := s.static.Trips[st.TripID]
		if !ok || !s.serviceActive(trip.ServiceID, now) {
			continue
		}

		sched := base.Add(time.Duration(st.Departure) * time.Second)
		if sched.Before(now.Add(-10*time.Minute)) || sched.After(now.Add(3*time.Hour)) {
			continue
		}

		route := s.static.Routes[trip.RouteID]
		headsign := st.Headsign
		if headsign == "" {
			headsign = trip.Headsign
		}

		dep := Departure{
			Line:          route.ShortName,
			RouteID:       route.ID,
			TripID:        trip.ID,
			Headsign:      headsign,
			ScheduledTime: sched.Format("15:04"),
			RealtimeTime:  sched.Format("15:04"),
		}

		if tu := rt.TripUpdates[trip.ID]; tu != nil {
			if realtime, delay, cancelled, ok := realtimeForStop(tu, stopID, sched); ok {
				dep.RealtimeTime = realtime.Format("15:04")
				dep.DelayMinutes = int(delay / 60)
				dep.Cancelled = cancelled
			}
		}

		if v := rt.Vehicles[trip.ID]; v != nil && sched.After(now) {
			dep.Vehicle = &Vehicle{
				ID: v.GetVehicle().GetId(),
			}
			if p := v.GetPosition(); p != nil {
				dep.Vehicle.Lat = p.GetLatitude()
				dep.Vehicle.Lon = p.GetLongitude()
			}
		}

		out = append(out, dep)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].RealtimeTime < out[j].RealtimeTime
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func realtimeForStop(tu *gtfs.TripUpdate, stopID string, scheduled time.Time) (time.Time, int32, bool, bool) {
	cancelled := tu.GetTrip().GetScheduleRelationship().String() == "CANCELED"

	for _, stu := range tu.GetStopTimeUpdate() {
		if stu.GetStopId() != stopID {
			continue
		}

		ev := stu.GetDeparture()
		if ev == nil {
			ev = stu.GetArrival()
		}
		if ev == nil {
			return scheduled, 0, cancelled, true
		}

		delay := ev.GetDelay()
		if ev.GetTime() != 0 {
			return time.Unix(ev.GetTime(), 0), delay, cancelled, true
		}

		return scheduled.Add(time.Duration(delay) * time.Second), delay, cancelled, true
	}

	return scheduled, 0, cancelled, false
}

func loadStatic(url string, cache *cache) (Static, error) {
	if err := cache.update(url); err != nil {
		return Static{}, err
	}

	zr, err := zip.NewReader(bytes.NewReader(cache.buffer), int64(len(cache.buffer)))
	if err != nil {
		return Static{}, err
	}

	st := Static{
		Stops:       map[string]Stop{},
		Routes:      map[string]Route{},
		Trips:       map[string]Trip{},
		StopTimes:   map[string][]StopTime{},
		Calendar:    map[string]Calendar{},
		DateChanges: map[string]map[string]int{},
	}

	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}

	readCSV(files["stops.txt"], func(row map[string]string) {
		st.Stops[row["stop_id"]] = Stop{
			ID:   row["stop_id"],
			Name: row["stop_name"],
		}
	})

	readCSV(files["routes.txt"], func(row map[string]string) {
		st.Routes[row["route_id"]] = Route{
			ID:        row["route_id"],
			ShortName: firstNonEmpty(row["route_short_name"], row["route_long_name"]),
			LongName:  row["route_long_name"],
		}
	})

	readCSV(files["trips.txt"], func(row map[string]string) {
		st.Trips[row["trip_id"]] = Trip{
			ID:        row["trip_id"],
			RouteID:   row["route_id"],
			ServiceID: row["service_id"],
			Headsign:  row["trip_headsign"],
		}
	})

	readCSV(files["stop_times.txt"], func(row map[string]string) {
		dep, err := parseGTFSTime(firstNonEmpty(row["departure_time"], row["arrival_time"]))
		if err != nil {
			return
		}

		stopID := row["stop_id"]
		st.StopTimes[stopID] = append(st.StopTimes[stopID], StopTime{
			TripID:    row["trip_id"],
			StopID:    stopID,
			Departure: dep,
			Headsign:  row["stop_headsign"],
		})
	})

	readCSV(files["calendar.txt"], func(row map[string]string) {
		var wd [7]bool
		wd[time.Monday] = row["monday"] == "1"
		wd[time.Tuesday] = row["tuesday"] == "1"
		wd[time.Wednesday] = row["wednesday"] == "1"
		wd[time.Thursday] = row["thursday"] == "1"
		wd[time.Friday] = row["friday"] == "1"
		wd[time.Saturday] = row["saturday"] == "1"
		wd[time.Sunday] = row["sunday"] == "1"

		st.Calendar[row["service_id"]] = Calendar{
			Weekdays: wd,
			Start:    parseDate(row["start_date"]),
			End:      parseDate(row["end_date"]),
		}
	})

	readCSV(files["calendar_dates.txt"], func(row map[string]string) {
		sid := row["service_id"]
		date := row["date"]
		ex, _ := strconv.Atoi(row["exception_type"])

		if st.DateChanges[sid] == nil {
			st.DateChanges[sid] = map[string]int{}
		}
		st.DateChanges[sid][date] = ex
	})

	for stopID := range st.StopTimes {
		sort.Slice(st.StopTimes[stopID], func(i, j int) bool {
			return st.StopTimes[stopID][i].Departure < st.StopTimes[stopID][j].Departure
		})
	}

	return st, nil
}

func readCSV(f *zip.File, fn func(map[string]string)) {
	if f == nil {
		return
	}

	rc, err := f.Open()
	if err != nil {
		return
	}
	defer rc.Close()

	r := csv.NewReader(rc)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		row := map[string]string{}
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		fn(row)
	}
}

func (s *Server) serviceActive(serviceID string, t time.Time) bool {
	key := t.Format("20060102")

	if change := s.static.DateChanges[serviceID][key]; change == 1 {
		return true
	} else if change == 2 {
		return false
	}

	cal, ok := s.static.Calendar[serviceID]
	if !ok {
		return false
	}

	d := midnight(t)
	if d.Before(cal.Start) || d.After(cal.End) {
		return false
	}

	return cal.Weekdays[t.Weekday()]
}

func parseGTFSTime(v string) (int, error) {
	p := strings.Split(v, ":")
	if len(p) != 3 {
		return 0, fmt.Errorf("bad GTFS time: %q", v)
	}

	h, _ := strconv.Atoi(p[0])
	m, _ := strconv.Atoi(p[1])
	s, _ := strconv.Atoi(p[2])

	return h*3600 + m*60 + s, nil
}

func parseDate(v string) time.Time {
	t, _ := time.ParseInLocation("20060102", v, time.Local)
	return t
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

var page = template.Must(template.New("index").Parse(`
<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>Departures</title>
	<style>
		body {
			font-family: sans-serif;
			max-width: 1100px;
			margin: 2rem auto;
			padding: 0 1rem;
		}
		table {
			width: 100%;
			border-collapse: collapse;
		}
		th, td {
			padding: .5rem;
			border-bottom: 1px solid #ddd;
			text-align: left;
		}
		.delay {
			color: #b00020;
			font-weight: bold;
		}
		.cancelled {
			text-decoration: line-through;
			color: #777;
		}
	</style>
</head>
<body>
	<h1>Departures for {{.StopID}}</h1>

	<p>
		Stop ID:
		<input id="stop" value="{{.StopID}}">
		<button onclick="load()">Load</button>
	</p>

	<table>
		<thead>
			<tr>
				<th>Line</th>
				<th>Destination</th>
				<th>Scheduled</th>
				<th>Realtime</th>
				<th>Delay</th>
				<th>Vehicle</th>
			</tr>
		</thead>
		<tbody id="rows"></tbody>
	</table>

<script>
async function load() {
	const stop = document.getElementById("stop").value;
	const res = await fetch("/api/departures?stop=" + encodeURIComponent(stop));
	const deps = await res.json();

	const rows = document.getElementById("rows");
	rows.innerHTML = "";

	for (const d of deps) {
		const tr = document.createElement("tr");
		if (d.cancelled) tr.className = "cancelled";

		let vehicle = "";
		if (d.vehicle) {
			const lat = d.vehicle.lat.toFixed(5);
			const lon = d.vehicle.lon.toFixed(5);
			vehicle = ` + "`<a target=\"_blank\" href=\"https://www.openstreetmap.org/?mlat=${lat}&mlon=${lon}#map=16/${lat}/${lon}\">${lat}, ${lon}</a>`" + `;
		}

		tr.innerHTML = ` + "`" + `
			<td>${esc(d.line)}</td>
			<td>${esc(d.headsign)}</td>
			<td>${esc(d.scheduled_time)}</td>
			<td>${esc(d.realtime_time)}</td>
			<td class="${d.delay_minutes > 0 ? "delay" : ""}">
				${d.cancelled ? "cancelled" : (d.delay_minutes ? "+" + d.delay_minutes + " min" : "")}
			</td>
			<td>${vehicle}</td>
		` + "`" + `;

		rows.appendChild(tr);
	}
}

function esc(s) {
	return String(s ?? "").replace(/[&<>"']/g, c => ({
		"&": "&amp;",
		"<": "&lt;",
		">": "&gt;",
		'"': "&quot;",
		"'": "&#39;"
	}[c]));
}

load();
setInterval(load, 15000);
</script>
</body>
</html>
`))
