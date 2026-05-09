<script lang="ts">
	import { onMount } from "svelte";

	let stop = new URL(document.location.toString()).searchParams.get("stop") ?? "";
	let showTerminal = new URL(document.location.toString()).searchParams.has("show-terminal")
		? new URL(document.location.toString()).searchParams.get("show-terminal") !== null
		: true;

	let stopInfo = null;
	let stopResults = [];
	let departures = [];
	let vehicles = [];

	let openTrip = null;
	let openTripInfo = null;
	let openTripStops = [];
	let openTripLoading = false;

	function result(json) {
		return json?.result ?? null;
	}

	function asList(v) {
		if (!v) return [];
		return Array.isArray(v) ? v : [v];
	}

	function hhmm(d) {
		return d.toLocaleTimeString("nl-NL", {
			hour: "2-digit",
			minute: "2-digit",
			hour12: false
		});
	}

	function delayText(minutes) {
		if (!minutes) return "0 min";
		return `${minutes > 0 ? "+" : ""}${minutes} min`;
	}

	function statusText(status) {
		switch (status) {
		case "DELAY": return "delayed";
		case "INIT": return "starting point";
		case "ARRIVAL": return "arriving";
		case "ONSTOP": return "stopping";
		case "DEPARTURE": return "passed";
		case "ONROUTE": return "near";
		case "OFFROUTE": return "off route";
		case "END": return "finished";
		default: return "";
		}
	}

	function lineStyle(d) {
		let style = "text-align: center;";
		if (d.route_text_color) style += `color: #${d.route_text_color};`;
		if (d.route_color) style += `background-color: #${d.route_color};`;
		return style;
	}

	function realtimeTime(d) {
		return (d.scheduled_time ?? 0) + (d.punctuality ?? 0);
	}

	function delayMinutes(d) {
		return Math.trunc((d.punctuality ?? 0) / 60);
	}

	function displayedDepartures(list) {
		return list.filter(d => showTerminal || !d.terminal);
	}

	function rowsWithMarkers(list, insertNow) {
		const out = [];
		let prevDate = "";
		let nowInserted = !insertNow;
		const now = new Date();

		for (const d of displayedDepartures(list)) {
			const departureTime = new Date(realtimeTime(d) * 1000);
			const date = departureTime.toLocaleDateString("nl-NL", {
				day: "2-digit",
				month: "2-digit",
				year: "numeric"
			});

			if (date !== prevDate) {
				prevDate = date;
				out.push({ type: "date", date });
			}

			if (!nowInserted && departureTime > now) {
				out.push({ type: "now" });
				nowInserted = true;
			}

			out.push({ type: "departure", departure: d });
		}

		if (!nowInserted)
			out.push({ type: "now" });

		return out;
	}

	function vehicleMapURL(v) {
		if (!v?.data_owner_code || !v?.vehicle_number) return "";
		return `/api/vehicle/${encodeURIComponent(v.data_owner_code)}/${encodeURIComponent(v.vehicle_number)}/map.png`;
	}

	function tripMapURL(tripID) {
		if (!tripID) return "";
		return `/api/trip/${encodeURIComponent(tripID)}/map.png`;
	}

	async function fetchJSON(url) {
		const res = await fetch(url);
		const json = await res.json();
		if (!res.ok) throw new Error(json?.error ?? res.statusText);
		return result(json);
	}

	async function loadStopInfo() {
		if (!stop) return;

		const data = await fetchJSON("/api/stop/" + encodeURIComponent(stop));
		const list = asList(data);
		stopInfo = list[0] ?? null;
	}

	async function loadDepartures() {
		if (!stop) return;

		const [depData, vehData] = await Promise.all([
			fetchJSON("/api/stop/" + encodeURIComponent(stop) + "/departures"),
			fetchJSON("/api/stop/" + encodeURIComponent(stop) + "/vehicles")
		]);

		departures = asList(depData);
		vehicles = asList(vehData);

		if (openTrip && !departures.some(d => d.trip_id === openTrip)) {
			openTrip = null;
			openTripInfo = null;
			openTripStops = [];
		}
	}

    async function updateTripCard() {
        if (!openTrip) return;

		const [info, stops] = await Promise.all([
			fetchJSON("/api/trip/" + encodeURIComponent(openTrip)),
			fetchJSON("/api/trip/" + encodeURIComponent(openTrip) + "/stops")
		]);

		openTripInfo = info;
		openTripStops = asList(stops);	
    }

	async function openTripCard(d) {
		if (openTrip === d.trip_id) {
			openTrip = null;
			openTripInfo = null;
			openTripStops = [];
			return;
		}

		openTrip = d.trip_id;
		openTripInfo = null;
		openTripStops = [];
		openTripLoading = true;

		try {
            await updateTripCard();
		} finally {
			openTripLoading = false;
		}
	}

	async function searchStops() {
		const q = stop.trim();
		if (q.length < 2) {
			stopResults = [];
			return;
		}

		const data = await fetchJSON("/api/stop_query?q=" + encodeURIComponent(q));
		stopResults = asList(data);
	}

	function selectStop(s) {
		stop = s.id;
		stopResults = [];
	}

	function submit() {
		const url = new URL(document.location.toString());
		url.searchParams.set("stop", stop);

		if (showTerminal)
			url.searchParams.set("show-terminal", "on");
		else
			url.searchParams.delete("show-terminal");

		document.location = url.toString();
	}

	onMount(() => {
		loadStopInfo();
		loadDepartures();

        const timer = setInterval(() => {
            if (openTrip) 
                updateTripCard();
            else
                loadDepartures();
        }, 10000); // 10 secounds
		return () => clearInterval(timer);
	});
</script>

<h1>
	{#if stopInfo}
		Departures for <i>{stopInfo.stop_name}</i>
	{:else}
		Departures
	{/if}
</h1>

<div class="controls">
	<form class="controls-form" on:submit|preventDefault={submit}>
		<div class="field-group">
			<div class="autocomplete">
				<input
					type="text"
					bind:value={stop}
					on:input={searchStops}
					placeholder="Search stop..."
					autocomplete="off"
				>
				{#if stopResults.length}
					<div class="autocomplete-results">
						{#each stopResults as s}
							<div on:click={() => selectStop(s)}>
								{s.name} ({s.id})
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>

		<div class="field-inline">
			<label>
				<input type="checkbox" bind:checked={showTerminal}>
				<span>Show terminal trips</span>
			</label>
		</div>

		<div class="actions">
			<input type="submit" value="Load">
		</div>
	</form>
</div>

<h2>Vehicles at stop</h2>

<table>
	<thead>
		<tr>
			<th style="width:20px;"></th>
			<th style="width:20px;"></th>
			<th>Destination</th>
			<th>Status</th>
			<th>Delay</th>
			<th>Vehicle</th>
		</tr>
	</thead>
	<tbody>
		{#each vehicles as v}
			<tr class="departure-row">
				<td style="text-align: center;">{v.platform_code ?? ""}</td>
				<td style={lineStyle(v)}>{v.route_short_name ?? ""}</td>
				<td>{v.trip_headsign ?? ""}</td>
				<td>{statusText(v.status)}</td>
				<td>{delayText(Math.trunc((v.punctuality ?? 0) / 60))}</td>
				<td>
					{#if vehicleMapURL(v)}
						<a target="_blank" href={vehicleMapURL(v)}>{v.vehicle_number}</a>
					{:else}
						{v.vehicle_number ?? ""}
					{/if}
				</td>
			</tr>
		{/each}
	</tbody>
</table>

<h2>Departures</h2>

<table>
	<thead>
		<tr>
			<th></th>
			<th style="width:20px;"></th>
			<th style="width:20px;"></th>
			<th>Destination</th>
			<th>Scheduled</th>
			<th>Delay</th>
		</tr>
	</thead>
	<tbody>
		{#each rowsWithMarkers(departures, true) as row}
			{#if row.type === "date"}
				<tr>
					<td colspan="6" class="date-row">{row.date}</td>
				</tr>
			{:else if row.type === "now"}
				<tr class="now-row">
					<td colspan="6"><div class="now-line"><span>nu</span></div></td>
				</tr>
			{:else}
				{@const d = row.departure}
				{@const scheduled = new Date(d.scheduled_time * 1000)}
				{@const realtime = new Date(realtimeTime(d) * 1000)}
				{@const delay = delayMinutes(d)}
				{@const open = openTrip === d.trip_id}

				<tr
					class:departure-row={true}
					class:has-next={true}
					class:terminal={d.terminal}
					class:alert-high={d.warning}
					class:open
					on:click={() => openTripCard(d)}
				>
					<td class="toggle-cell">{open ? "▾" : "▸"}</td>
					<td style="text-align: center;">{d.platform_code ?? ""}</td>
					<td style={lineStyle(d)}>{d.route_short_name}</td>
					<td>{d.headsign}</td>
					<td>
						{#if d.scheduled_time === realtimeTime(d)}
							<b>{hhmm(scheduled)}</b>
						{:else}
							<s>{hhmm(scheduled)}</s> <b>{hhmm(realtime)}</b>
						{/if}
					</td>
					<td class:delay={delay > 0}>
						{#if delay !== 0}
							{delay > 0 ? "+" : ""}{delay} min
						{/if}
					</td>
				</tr>

				{#if open}
					<tr class="trip-card-row">
						<td colspan="6">
							<div class="trip-card">
								{#if openTripLoading}
									Loading trip...
								{:else if openTripInfo}
									<div class="trip-card-header">
										<div>
											<b>{openTripInfo.route_short_name ?? ""}</b>
											{openTripInfo.trip_headsign ?? ""}
											<div class="muted">
												trip {openTripInfo.trip_id}
												{#if openTripInfo.vehicle_number}
													— vehicle {openTripInfo.vehicle_number}
												{/if}
												{#if openTripInfo.block_code}
													— block {openTripInfo.block_code}
												{/if}
												{#if openTripInfo.status}
													— {statusText(openTripInfo.status)}
												{/if}
											</div>
										</div>

										<div class="trip-card-links">
											<a target="_blank" href={tripMapURL(openTripInfo.trip_id)}>trip map</a>
											{#if openTripInfo.data_owner_code && openTripInfo.vehicle_number}
												<a target="_blank" href={vehicleMapURL(openTripInfo)}>vehicle map</a>
											{/if}
										</div>
									</div>

									{#if vehicleMapURL(openTripInfo)}
										<img
											class="trip-map"
											src={vehicleMapURL(openTripInfo)}
											alt="Vehicle map"
										>
									{:else}
										<img
											class="trip-map"
											src={tripMapURL(openTripInfo.trip_id)}
											alt="Trip map"
										>
									{/if}

									<table class="trip-stops">
										<thead>
											<tr>
												<th>#</th>
												<th>Stop</th>
												<th>Arrival</th>
												<th>Departure</th>
												<th>Status</th>
												<th>Delay</th>
											</tr>
										</thead>
										<tbody>
											{#each openTripStops as st}
												<tr class:current-stop={st.status && st.status !== "END"}>
													<td>{st.stop_sequence}</td>
													<td>
														{st.stop_name}
														{#if st.platform_code}
															<span class="muted">platform {st.platform_code}</span>
														{/if}
													</td>
													<td>{st.arrival_time ?? ""}</td>
													<td>{st.departure_time ?? ""}</td>
													<td>{statusText(st.status)}</td>
													<td>{delayText(Math.trunc((st.punctuality ?? 0) / 60))}</td>
												</tr>
											{/each}
										</tbody>
									</table>
								{/if}
							</div>
						</td>
					</tr>
				{/if}
			{/if}
		{/each}
	</tbody>
</table>

<footer class="footer">
	&copy; 2026 Friedel Schön &mdash;
	Source hosted at
	<a href="https://github.com/friedelschoen/departures" target="_blank">
		GitHub
	</a>
</footer>
