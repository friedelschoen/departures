WITH q AS (
	SELECT
		lower($1) AS raw,
		'%' || lower($1) || '%' AS like
),
results AS (
	SELECT *
	FROM (
		SELECT
			'stop' AS type,
			s.stop_name AS label,
			'stop ' || s.stop_id AS subtitle,
			s.stop_id,
			NULL::text AS route_id,
			NULL::text AS data_owner_code,
			NULL::integer AS vehicle_number,
			NULL::integer AS block_code,
			NULL::text AS route_short_name,
			NULL::text AS route_color,
			NULL::text AS route_text_color,
			10 AS rank
		FROM active_gtfs_stops s, q
		WHERE lower(s.stop_name) LIKE q.like
		   OR lower(s.stop_id) = q.raw
		   OR lower(s.stop_code) = q.raw
		LIMIT 15
	) stops

	UNION ALL

	SELECT *
	FROM (
		SELECT DISTINCT ON (r.route_id)
			'route' AS type,
			'Line ' || COALESCE(r.route_short_name, r.route_id) AS label,
			COALESCE(r.route_long_name, 'route ' || r.route_id) AS subtitle,
			NULL::text AS stop_id,
			r.route_id,
			NULL::text AS data_owner_code,
			NULL::integer AS vehicle_number,
			NULL::integer AS block_code,
			r.route_short_name,
			r.route_color,
			r.route_text_color,
			20 AS rank
		FROM active_gtfs_routes r, q
		WHERE lower(r.route_short_name) LIKE q.like
		   OR lower(r.route_long_name) LIKE q.like
		   OR lower(r.route_id) = q.raw
		ORDER BY r.route_id
		LIMIT 15
	) routes

	UNION ALL

	SELECT *
	FROM (
		SELECT DISTINCT ON (v.data_owner_code, v.vehicle_number)
			'vehicle' AS type,
			'Vehicle ' || v.vehicle_number::text AS label,
			v.data_owner_code AS subtitle,
			NULL::text AS stop_id,
			NULL::text AS route_id,
			v.data_owner_code,
			v.vehicle_number,
			NULL::integer AS block_code,
			r.route_short_name,
			r.route_color,
			r.route_text_color,
			30 AS rank
		FROM kv6_current_vehicle v
		LEFT JOIN active_gtfs_trips t
			ON t.realtime_trip_id = v.realtime_trip_id
		   AND t.realtime_trip_sequence = 1
		LEFT JOIN active_gtfs_routes r
			ON r.feed_ref = t.feed_ref
		   AND r.route_id = t.route_id,
		q
		WHERE v.vehicle_number::text LIKE q.raw || '%'
		   OR lower(v.data_owner_code) LIKE q.like
		ORDER BY v.data_owner_code, v.vehicle_number
		LIMIT 15
	) vehicles

	UNION ALL

	SELECT *
	FROM (
		SELECT DISTINCT ON (h.data_owner_code, h.block_code)
			'block' AS type,
			'Omloop ' || h.block_code::text AS label,
			h.data_owner_code AS subtitle,
			NULL::text AS stop_id,
			NULL::text AS route_id,
			h.data_owner_code,
			NULL::integer AS vehicle_number,
			h.block_code,
			NULL::text AS route_short_name,
			NULL::text AS route_color,
			NULL::text AS route_text_color,
			40 AS rank
		FROM kv6_block_trip_history h, q
		WHERE h.operating_day = current_date
		  AND (
			h.block_code::text LIKE q.raw || '%'
			OR lower(h.data_owner_code) LIKE q.like
		  )
		ORDER BY h.data_owner_code, h.block_code, h.last_seen DESC
		LIMIT 15
	) blocks
)
SELECT *
FROM results
ORDER BY
	rank,
	label
LIMIT 50;
