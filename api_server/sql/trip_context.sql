WITH current_trip AS (
	SELECT
		h.operating_day,
		h.data_owner_code,
		h.block_code,
		t.feed_ref,
		t.service_id,
		t.trip_id
	FROM kv6_block_trip_history h

	JOIN active_gtfs_trips t
		ON t.realtime_trip_id = h.realtime_trip_id
	   AND t.realtime_trip_sequence = 1

	JOIN active_gtfs_calendar_dates cd
		ON cd.feed_ref = t.feed_ref
	   AND cd.service_id = t.service_id
	   AND cd.date = h.operating_day
	   AND cd.exception_type = 1

	WHERE t.trip_id = $1

	ORDER BY
		h.operating_day DESC,
		h.last_seen DESC

	LIMIT 1
),
block_trips AS (
	SELECT
		t.feed_ref,
		t.service_id,
		t.trip_id,
		t.trip_headsign,

		r.route_id,
		r.route_short_name,
		r.route_color,
		r.route_text_color,

		tb.start_time,
		tb.end_time,

		lag(t.trip_id) OVER w AS previous_trip_id,
		lead(t.trip_id) OVER w AS next_trip_id

	FROM current_trip cur

	JOIN kv6_block_trip_history h
		ON h.operating_day = cur.operating_day
	   AND h.data_owner_code = cur.data_owner_code
	   AND h.block_code = cur.block_code

	JOIN active_gtfs_trips t
		ON t.feed_ref = cur.feed_ref
	   AND t.realtime_trip_id = h.realtime_trip_id
	   AND t.realtime_trip_sequence = 1

	JOIN active_gtfs_calendar_dates cd
		ON cd.feed_ref = t.feed_ref
	   AND cd.service_id = t.service_id
	   AND cd.date = h.operating_day
	   AND cd.exception_type = 1

	JOIN active_gtfs_routes r
		ON r.feed_ref = t.feed_ref
	   AND r.route_id = t.route_id

	JOIN active_gtfs_trip_bounds tb
		ON tb.feed_ref = t.feed_ref
	   AND tb.trip_id = t.trip_id

	WINDOW w AS (
		ORDER BY
			tb.start_time,
			tb.end_time,
			t.trip_id
	)
),
ctx AS (
	SELECT *
	FROM block_trips
	WHERE trip_id = $1
	LIMIT 1
)
SELECT
	CASE
		WHEN t.trip_id = ctx.previous_trip_id THEN 'previous'
		WHEN t.trip_id = ctx.next_trip_id THEN 'next'
	END AS relation,

	t.trip_id,
	t.trip_headsign,
	t.route_id,
	t.route_short_name,
	t.route_color,
	t.route_text_color

FROM ctx

JOIN block_trips t
	ON t.trip_id IN (ctx.previous_trip_id, ctx.next_trip_id)

ORDER BY
	CASE
		WHEN t.trip_id = ctx.previous_trip_id THEN 0
		WHEN t.trip_id = ctx.next_trip_id THEN 1
	END;
