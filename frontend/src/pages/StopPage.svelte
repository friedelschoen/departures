<script lang="ts">
    import { onMount } from "svelte";
    import MapView from "../components/MapView.svelte";
    import type { Stop } from "../schema/Stop";
    import type { Trip } from "../schema/Trip";
    import type { TripStop } from "../schema/TripStop";
    import type { Vehicle } from "../schema/Vehicle";
    import type { StopDeparture } from "../schema/StopDeparture";
    import type { StopArrival } from "../schema/StopArrival";
    import type { StopVehicle } from "../schema/StopVehicle";
    import type { StopQuery } from "../schema/StopQuery";
    import type { StopRoute } from "../schema/StopRoute";
    import TripCard from "../components/TripCard.svelte";
    import { vehicleURL } from "../lib/links";

    export let stop: string;

    let url = new URL(document.location.toString());
    let mode = url.searchParams.get("mode") ?? "departure";
    if (mode !== "arrival") mode = "departure";
    let showTerminal = url.searchParams.get("show-terminal") == "on";

    let stopInfo: Stop | null = null;
    let events: StopDeparture[] | StopArrival[] = [];
    let vehicles: StopVehicle[] = [];
    let routes: StopRoute[] = [];

    let query: StopQuery[] = [];

    let openTrip: string | null = null;

    let openVehicle: string | null = null;
    let openVehicleInfo: Vehicle | null = null;
    let openVehicleLoading = false;

    function hhmm(d: Date) {
        return d.toLocaleTimeString("nl-NL", {
            hour: "2-digit",
            minute: "2-digit",
            hour12: false,
        });
    }

    function delayText(minutes: number) {
        if (!minutes) return "0 min";
        return `${minutes > 0 ? "+" : ""}${minutes} min`;
    }

    function vehicleWaitText(minutes: number) {
        if (minutes <= 0) return `${-minutes} min`;
        return `+${minutes} min`;
    }

    function statusText(status: string): string {
        switch (status) {
            case "DELAY":
                return "delayed";
            case "INIT":
                return "starting point";
            case "ARRIVAL":
                return "arriving";
            case "ONSTOP":
                return "stopping";
            case "DEPARTURE":
                return "passed";
            case "ONROUTE":
                return "near";
            case "OFFROUTE":
                return "off route";
            case "END":
                return "finished";
            default:
                return "";
        }
    }

    function lineStyle(
        d: StopVehicle | StopDeparture | StopArrival | StopRoute,
    ) {
        let style = "text-align: center;";
        if (d.route_text_color) style += `color: #${d.route_text_color};`;
        if (d.route_color) style += `background-color: #${d.route_color};`;
        return style;
    }

    function delayMinutes(d: { punctuality: any }) {
        return Math.trunc((d.punctuality ?? 0) / 60);
    }

    function rowsWithMarkers(
        list: StopDeparture[] | StopArrival[],
        insertNow: boolean,
    ): (
        | { type: "date"; date: string }
        | { type: "now" }
        | { type: "departure"; departure: StopDeparture | StopArrival }
    )[] {
        const out: (
            | { type: "date"; date: string }
            | { type: "now" }
            | { type: "departure"; departure: StopDeparture | StopArrival }
        )[] = [];
        let prevDate = "";
        let nowInserted = !insertNow;
        const now = new Date();

        for (const d of list) {
            if (d.terminal && !showTerminal) continue;
            const departureTime = new Date(
                d.scheduled_time * 1000,
            ); /* NOTE: relative to sorting */
            const date = departureTime.toLocaleDateString("nl-NL", {
                day: "2-digit",
                month: "2-digit",
                year: "numeric",
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

        if (!nowInserted) out.push({ type: "now" });

        return out;
    }

    function vehicleMapURL(v: StopVehicle | Vehicle | Trip) {
        if (!v?.data_owner_code || !v?.vehicle_number) return "";
        return `/api/vehicle/${encodeURIComponent(v.data_owner_code)}/${encodeURIComponent(v.vehicle_number)}/map.png`;
    }

    function tripMapURL(tripID: string | number | boolean | undefined) {
        if (!tripID) return "";
        return `/api/trip/${encodeURIComponent(tripID)}/map.png`;
    }

    async function fetchJSON(url: URL | RequestInfo) {
        const res = await fetch(url);
        const json = await res.json();
        if (!res.ok) throw new Error(json?.error ?? res.statusText);
        return json.result;
    }

    async function loadStopInfo() {
        if (!stop) return;

        const data = await fetchJSON("/api/stop/" + encodeURIComponent(stop));
        const list = data;
        stopInfo = list[0] ?? null;
    }

    async function loadEvents() {
        if (!stop) return;

        const eventPath = mode === "arrival" ? "arrivals" : "departures";

        [events, vehicles, routes] = await Promise.all([
            fetchJSON(
                "/api/stop/" + encodeURIComponent(stop) + "/" + eventPath,
            ),
            fetchJSON("/api/stop/" + encodeURIComponent(stop) + "/vehicles"),
            fetchJSON("/api/stop/" + encodeURIComponent(stop) + "/routes"),
        ]);
    }

    async function openVehicleCard(vehicle: StopVehicle) {
        const dataOwner = vehicle.data_owner_code;
        const vehicleNumber = vehicle.vehicle_number;
        const key = `${dataOwner}/${vehicleNumber}`;

        if (openVehicle === key) {
            openVehicle = null;
            openVehicleInfo = null;
            return;
        }

        openVehicle = key;
        openVehicleInfo = null;
        openVehicleLoading = true;

        try {
            const data = await fetchJSON(
                `/api/vehicle/${encodeURIComponent(dataOwner)}/${encodeURIComponent(vehicleNumber)}`,
            );
            openVehicleInfo = data[0] ?? null;
        } finally {
            openVehicleLoading = false;
        }
    }

    onMount(() => {
        loadStopInfo();
        loadEvents();

        const timer = setInterval(() => {
            loadEvents();
        }, 15000); // 15 secounds
        return () => clearInterval(timer);
    });
</script>

<h1>
    {mode === "arrival" ? "Arrivals" : "Departures"}
    {#if stopInfo}
        for <i>{stopInfo.stop_name}</i>
    {/if}
</h1>

<h2>Routes</h2>

{#if routes.length}
    <div class="route-list">
        {#each routes as r}
            <a
                class="route-pill"
                style={lineStyle(r)}
                href={`/route/${encodeURIComponent(r.route_id)}`}
            >
                {r.route_short_name ?? r.route_id}
            </a>
        {/each}
    </div>
{:else}
    <div class="muted">No routes found.</div>
{/if}

<h2>Vehicles</h2>

<table>
    <thead>
        <tr>
            <th></th>
            <th style="width:20px;"></th>
            <th style="width:20px;"></th>
            <th>Destination</th>
            <th>Departing in </th>
            <th>Vehicle</th>
        </tr>
    </thead>
    <tbody>
        {#each vehicles as v}
            {@const key = `${v.data_owner_code}/${v.vehicle_number}`}
            {@const open = openVehicle === key}
            <tr
                class:departure-row={true}
                class:has-next={true}
                class:open
                on:click={() => openVehicleCard(v)}
            >
                <td class="toggle-cell">{open ? "▾" : "▸"}</td>
                <td style="text-align: center;">{v.platform_code ?? ""}</td>
                <td style={lineStyle(v)}>{v.route_short_name ?? ""}</td>
                <td>{v.trip_headsign ?? ""}</td>
                <td>{vehicleWaitText(Math.trunc((v.punctuality ?? 0) / 60))}</td
                >
                <td>
                    <a href={vehicleURL(v.data_owner_code, v.vehicle_number)}>
                        {v.vehicle_number}
                    </a>
                </td>
            </tr>
            {#if open}
                <tr class="trip-card-row">
                    <td colspan="7">
                        <div class="trip-card">
                            {#if openVehicleLoading}
                                Loading vehicle...
                            {:else if openVehicleInfo}
                                <div class="trip-card-header">
                                    <div>
                                        <b
                                            >Vehicle {openVehicleInfo.vehicle_number}</b
                                        >
                                        {#if openVehicleInfo.route_short_name}
                                            — line {openVehicleInfo.route_short_name}
                                        {/if}
                                        <div class="muted">
                                            {#if openVehicleInfo.trip_headsign}
                                                {openVehicleInfo.trip_headsign}
                                            {/if}
                                            {#if openVehicleInfo.realtime_trip_id}
                                                — realtime trip {openVehicleInfo.realtime_trip_id}
                                            {/if}
                                            {#if openVehicleInfo.block_code}
                                                — block {openVehicleInfo.block_code}
                                            {/if}
                                            {#if openVehicleInfo.status}
                                                — {statusText(
                                                    openVehicleInfo.status,
                                                )}
                                            {/if}
                                        </div>
                                    </div>
                                </div>
                                <MapView
                                    tripID={openVehicleInfo.trip_id}
                                    vehicle={openVehicleInfo}
                                    focus="vehicle"
                                />
                            {:else}
                                No vehicle data found.
                            {/if}
                        </div>
                    </td>
                </tr>
            {/if}
        {/each}
    </tbody>
</table>

<h2>
    {mode === "arrival" ? "Arrivals" : "Departures"}
    <button
        class:mode-switch={true}
        on:click={() => (mode = mode === "arrival" ? "departure" : "arrival")}
        >&#8644;
        {mode === "arrival" ? "Departures" : "Arrivals"}
    </button>
</h2>

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
                    <td colspan="6"
                        ><div class="now-line"><span>nu</span></div></td
                    >
                </tr>
            {:else}
                {@const d = row.departure}
                {@const scheduled = new Date(d.scheduled_time * 1000)}
                {@const realtime = new Date(
                    (d.scheduled_time + d.punctuality) * 1000,
                )}
                {@const delay = delayMinutes(d)}
                {@const open = openTrip === d.trip_id}

                <tr
                    class:departure-row={true}
                    class:has-next={true}
                    class:terminal={d.terminal}
                    class:alert-high={d.warning}
                    class:open
                    on:click={() => (openTrip = d.trip_id)}
                >
                    <td class="toggle-cell">{open ? "▾" : "▸"}</td>
                    <td style="text-align: center;">{d.platform_code ?? ""}</td>
                    <td style={lineStyle(d)}>{d.route_short_name}</td>
                    <td class:at-stop={d.at_stop}>{d.headsign}</td>
                    <td>
                        {#if d.punctuality == 0}
                            <b>{hhmm(scheduled)}</b>
                        {:else}
                            <s>{hhmm(scheduled)}</s> <b>{hhmm(realtime)}</b>
                        {/if}
                    </td>
                    <td class:delay={delay > 0}>
                        {#if d.status}
                            {delayText(delay)}
                        {/if}
                    </td>
                </tr>

                {#if open && openTrip}
                    <tr class="trip-card-row">
                        <td colspan="6">
                            <TripCard trip={openTrip} />
                        </td>
                    </tr>
                {/if}
            {/if}
        {/each}
    </tbody>
</table>
