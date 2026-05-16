<script lang="ts">
    import { vehicleURL, blockURL, routeURL } from "../lib/links";
    import MapView from "./MapView.svelte";
    import TripStopTable from "./TripStopTable.svelte";

    import type { Trip } from "../schema/Trip";

    export let trip: string;

    let info: Trip | null = null;
    let loading = false;

    type TripContext = {
        relation: "previous" | "next";
        trip_id: string;
        trip_headsign: string | null;
        route_id: string;
        route_short_name: string | null;
        route_color: string | null;
        route_text_color: string | null;
    };

    let context: TripContext[] = [];

    $: previous = context.find((c) => c.relation === "previous") ?? null;
    $: next = context.find((c) => c.relation === "next") ?? null;

    async function load() {
        /*info = null;
        stops = [];
        context = [];*/
        loading = true;

        try {
            [info, context] = await Promise.all([
                fetchJSON("/api/trip/" + encodeURIComponent(trip)),
                fetchJSON("/api/trip/" + encodeURIComponent(trip) + "/context"),
            ]);
        } finally {
            loading = false;
        }
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

    function lineStyle(t: TripContext): string {
        let style = "";
        if (t.route_text_color) style += `color: #${t.route_text_color};`;
        if (t.route_color) style += `background-color: #${t.route_color};`;
        return style;
    }

    async function fetchJSON(url: URL | RequestInfo) {
        const res = await fetch(url);
        const json = await res.json();
        if (!res.ok) throw new Error(json?.error ?? res.statusText);
        return json.result;
    }

    let loadedTrip = "";

    $: if (trip && trip !== loadedTrip) {
        loadedTrip = trip;
        load();
    }
</script>

<div class="trip-card">
    {#if loading}
        <b>Loading trip...</b>
    {/if}
    {#if info}
        <div class="trip-card-header">
            <div class="trip-card-header">
                <div>
                    <b>{info.route_short_name ?? ""}</b>
                    {info.trip_headsign ?? ""}
                    <div class="muted">
                        trip {info.trip_id}

                        {#if info.route_id}
                            — <a href={routeURL(info.route_id)}
                                >route {info.route_short_name ??
                                    info.route_id}</a
                            >
                        {/if}

                        {#if info.vehicle_number}
                            — <a
                                href={vehicleURL(
                                    info.data_owner_code,
                                    info.vehicle_number,
                                )}
                            >
                                vehicle {info.vehicle_number}
                            </a>
                        {/if}

                        {#if info.block_code}
                            — <a
                                href={blockURL(
                                    info.data_owner_code,
                                    info.block_code,
                                )}
                            >
                                block {info.block_code}
                            </a>
                        {/if}

                        {#if info.status}
                            — {statusText(info.status)}
                        {/if}
                    </div>
                </div>
            </div>
        </div>

        <div class="trip-context">
            {#if previous}
                <button
                    class="link-button"
                    on:click={() => (trip = previous.trip_id)}
                >
                    ← <span
                        class="route-pill small"
                        style={lineStyle(previous)}
                    >
                        {previous.route_short_name ?? previous.route_id}
                    </span>
                    {previous.trip_headsign ?? previous.trip_id}
                </button>
            {:else}
                <span class="disabled">← previous</span>
            {/if}

            <span class="current">current</span>

            {#if next}
                <button
                    class="link-button"
                    on:click={() => (trip = next.trip_id)}
                >
                    <span class="route-pill small" style={lineStyle(next)}>
                        {next.route_short_name ?? next.route_id}
                    </span>
                    {next.trip_headsign ?? next.trip_id} →
                </button>
            {:else}
                <span class="disabled">next →</span>
            {/if}
        </div>

        <MapView tripID={trip} vehicle={info} />

        <div class="trip-card-links">
            <a href={`/trip/${encodeURIComponent(trip)}`}>open trip details</a>
        </div>
    {:else if !loading}
        No trip data found.
    {/if}
</div>
