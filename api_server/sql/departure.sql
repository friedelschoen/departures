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

WITH active_feed AS (
	SELECT id
	FROM gtfs_feeds
	WHERE active = true
	ORDER BY imported_at DESC
	LIMIT 1
),
vehicle_context AS (
	SELECT DISTINCT ON (
		vk.operating_day,
		vk.data_owner_code,
		vk.vehicle_number
	)
		vk.operating_day,
		vk.data_owner_code,
		vk.vehicle_number,

		vs.stop_id,
		vs.stop_name,

		vr.route_id,
		vr.route_short_name,
		vr.route_color,
		vr.route_text_color,

		vt.trip_id,
		vt.trip_headsign,

		vk.status,
        vk.timestamp,
		COALESCE(vk.punctuality, 0) AS punctuality,
		vk.block_code,
		vk.rd_x,
		vk.rd_y,

		vst.stop_sequence,
		vtb.start_sequence
	FROM kv6_journey_status vk
	JOIN active_feed af ON true
	JOIN gtfs_trips vt
		ON vt.feed_ref = af.id
	   AND vt.realtime_trip_id = vk.journey_key
	   AND vt.realtime_trip_sequence = 1
	JOIN gtfs_calendar_dates vcd
		ON vcd.feed_ref = vt.feed_ref
	   AND vcd.service_id = vt.service_id
	   AND vcd.date = vk.operating_day
	   AND vcd.exception_type = 1
	JOIN gtfs_routes vr
		ON vr.feed_ref = vt.feed_ref
	   AND vr.route_id = vt.route_id
	JOIN gtfs_trip_bounds vtb
		ON vtb.feed_ref = vt.feed_ref
	   AND vtb.trip_id = vt.trip_id
	JOIN gtfs_stops vs
		ON vs.feed_ref = vt.feed_ref
	   AND vs.stop_code = vk.user_stop_code
	JOIN gtfs_stop_times vst
		ON vst.feed_ref = vt.feed_ref
	   AND vst.trip_id = vt.trip_id
       AND vst.stop_id = vs.stop_id
	WHERE vk.vehicle_number IS NOT NULL
	ORDER BY
		vk.operating_day,
		vk.data_owner_code,
		vk.vehicle_number,
		vk.timestamp DESC,
		vst.stop_sequence
)
SELECT
	r.route_id,
	r.route_short_name,
	r.route_color,
	r.route_text_color,

	t.trip_id,
	COALESCE(NULLIF(st.stop_headsign, ''), t.trip_headsign) AS headsign,
	s.platform_code,

	EXTRACT(EPOCH FROM (
		(cd.date::timestamp + st.departure_time)
			AT TIME ZONE a.agency_timezone
	))::bigint AS scheduled_time,

	st.stop_sequence = tb.end_sequence,

	k.status,
    EXTRACT(EPOCH FROM k.timestamp)::bigint,
	CASE 
		WHEN vc.stop_sequence = vc.start_sequence 
		THEN GREATEST(COALESCE(k.punctuality, 0), 0)
		ELSE COALESCE(k.punctuality, 0)
	END AS final_punctuality,

	k.vehicle_number,
	k.block_code,
	k.rd_x,
	k.rd_y,

	vc.stop_id,
	vc.stop_name,

	vc.route_id,
	vc.route_short_name,
	vc.route_color,
	vc.route_text_color,

	vc.trip_id,
	vc.trip_headsign,

	vc.status,
    EXTRACT(EPOCH FROM vc.timestamp)::bigint,
	COALESCE(vc.punctuality, 0),
	vc.block_code,
	vc.rd_x,
	vc.rd_y

FROM active_feed af
JOIN gtfs_stop_times st
	ON st.feed_ref = af.id
JOIN gtfs_stops s
	ON s.feed_ref = st.feed_ref
   AND s.stop_id = st.stop_id
JOIN gtfs_trips t
	ON t.feed_ref = st.feed_ref
   AND t.trip_id = st.trip_id
JOIN gtfs_routes r
	ON r.feed_ref = t.feed_ref
   AND r.route_id = t.route_id
JOIN gtfs_agency a
	ON a.feed_ref = r.feed_ref
   AND a.agency_id = r.agency_id
JOIN gtfs_calendar_dates cd
	ON cd.feed_ref = t.feed_ref
   AND cd.service_id = t.service_id
   AND cd.exception_type = 1
   AND cd.date BETWEEN
		((now() AT TIME ZONE a.agency_timezone)::date - 1)
		AND
		((now() AT TIME ZONE a.agency_timezone)::date + 1)
JOIN gtfs_trip_bounds tb
	ON tb.feed_ref = st.feed_ref
   AND tb.trip_id = st.trip_id
LEFT JOIN kv6_journey_status k
	ON k.operating_day = cd.date
   AND k.journey_key = t.realtime_trip_id
LEFT JOIN vehicle_context vc
	ON vc.operating_day = k.operating_day
   AND vc.data_owner_code = k.data_owner_code
   AND vc.vehicle_number = k.vehicle_number
WHERE (s.stop_id = $1 OR s.parent_station = $1)
  AND (
		(cd.date::timestamp + st.departure_time)
			AT TIME ZONE a.agency_timezone
	  ) BETWEEN now() - $2::INTERVAL
	        AND now() + $3::INTERVAL
ORDER BY
	((cd.date::timestamp + st.departure_time)
		AT TIME ZONE a.agency_timezone) ASC;
