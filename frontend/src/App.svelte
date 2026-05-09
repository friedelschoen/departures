<script lang="ts">
	import { onMount } from "svelte";

    let url = new URL(document.location.toString());
	let stop = url.searchParams.get("stop") ?? "";
    let mode = url.searchParams.get("mode") ?? "departure";
    if (mode !== "arrival")
	    mode = "departure";
    let showTerminal = url.searchParams.get("show-terminal") == "on";

	let stopInfo = null;
	let stopResults = [];
	let events = [];
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

	function displayedEvents(list) {
		return list.filter(d => showTerminal || !d.terminal);
	}

	function rowsWithMarkers(list, insertNow) {
		const out = [];
		let prevDate = "";
		let nowInserted = !insertNow;
		const now = new Date();

		for (const d of displayedEvents(list)) {
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

    async function loadEvents() {
	    if (!stop) return;

    	const eventPath = mode === "arrival" ? "arrivals" : "departures";

	    const [eventData, vehData] = await Promise.all([
		    fetchJSON("/api/stop/" + encodeURIComponent(stop) + "/" + eventPath),
	    	fetchJSON("/api/stop/" + encodeURIComponent(stop) + "/vehicles")
    	]);

	    events = asList(eventData);
	    vehicles = asList(vehData);

    	if (openTrip && !events.some(d => d.trip_id === openTrip)) {
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
		stop = s.stop_id;
		stopResults = [];
	}

    function setMode(next) {
	    if (mode === next) return;

    	mode = next;
	    openTrip = null;
	    openTripInfo = null;
	    openTripStops = [];

    	const url = new URL(document.location.toString());
	    url.searchParams.set("mode", mode);
    	history.replaceState(null, "", url.toString());

    	loadEvents();
    }

function eventHHMM(v) {
	if (!v) return "";

	const d = new Date(v);
	if (!Number.isNaN(d.getTime()))
		return hhmm(d);

	if (typeof v === "number")
		return hhmm(new Date(v * 1000));

	return String(v).slice(11, 16);
}

function tripStopRows(stops) {
	const out = [];

	for (const st of stops) {
		const arr = st.arrival_time;
		const dep = st.departure_time;

		if (arr && dep && eventHHMM(arr) !== eventHHMM(dep)) {
			out.push({ ...st, event_mode: "A", event_time: arr });
			out.push({ ...st, event_mode: "D", event_time: dep });
		} else {
			out.push({
				...st,
				event_mode: arr ? "A" : "D",
				event_time: arr ?? dep
			});
		}
	}

	return out;
}

function stopRowClass(st) {
	const s = statusText(st.status);

	return {
		"near-stop": s === "near",
		"active-stop": s !== "" && s !== "passed" && s !== "near"
	};
}

    function submit() {
	    const url = new URL(document.location.toString());
    	url.searchParams.set("stop", stop);
	    url.searchParams.set("mode", mode);

    	if (showTerminal)
	    	url.searchParams.set("show-terminal", "on");
	    else
		    url.searchParams.delete("show-terminal");

    	document.location = url.toString();
    }

    onMount(() => {
		loadStopInfo();
		loadEvents();

        const timer = setInterval(() => {
            if (openTrip) 
                updateTripCard();
            else
                loadEvents();
        }, 10000); // 10 secounds
		return () => clearInterval(timer);
	});
</script>

<h1>
    {mode === "arrival" ? "Arrivals" : "Departures"}
	{#if stopInfo}
		for <i>{stopInfo.stop_name}</i>
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
								{s.stop_name} ({s.stop_id})
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

    <div class="tabs">
	<button
		type="button"
		class:active={mode === "departure"}
		on:click={() => setMode("departure")}
	>
		Departures
	</button>

	<button
		type="button"
		class:active={mode === "arrival"}
		on:click={() => setMode("arrival")}
	>
		Arrivals
	</button>
</div>
</div>

<h2>Vehicles</h2>

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

<h2>{mode === "arrival" ? "Arrivals" : "Departures"}</h2>

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
		{#each rowsWithMarkers(events, true) as row}
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
						{#if d.punctuality == 0}
							<b>{hhmm(scheduled)}</b>
						{:else}
							<s>{hhmm(scheduled)}</s> <b>{hhmm(realtime)}</b>
						{/if}
					</td>
					<td class:delay={delay > 0}>
						{delayText(delay)}
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
                                            <th></th>
                                            <th>Stop</th>
                                            <th>Time</th>
                                            <th>Delay</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each tripStopRows(openTripStops) as st}
                                            {@const s = statusText(st.status)}
                                            {@const delay = Math.trunc((st.punctuality ?? 0) / 60)}

                                            <tr
                                                class:near-stop={s === "near"}
                                                class:active-stop={s !== "" && s !== "passed" && s !== "near"}
                                            >
                                                <td>{st.stop_sequence}</td>
                                                <td class="event-mode">{st.event_mode}</td>
                                                <td>
                                                    {st.stop_name}
                                                    {#if st.platform_code}
                                                        <span class="muted">platform {st.platform_code}</span>
                                                    {/if}
                                                </td>
                                                <td>{eventHHMM(st.event_time)}</td>
                                                <td class="trip-delay">
                                                    {#if st.status}
                                                        {delayText(delay)}
                                                    {/if}
                                                </td>
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
