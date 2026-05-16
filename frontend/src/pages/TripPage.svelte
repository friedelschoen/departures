<script lang="ts">
    import TripCard from "../components/TripCard.svelte";
    import TripStopTable from "../components/TripStopTable.svelte";
    import type { TripStop } from "../schema/TripStop";

    export let trip: string;

    let stops: TripStop[] = [];
    let loadingStops = false;
    let loadedTrip = "";

    async function fetchJSON(url: URL | RequestInfo) {
        const res = await fetch(url);
        const json = await res.json();
        if (!res.ok) throw new Error(json?.error ?? res.statusText);
        return json.result;
    }

    async function loadStops(currentTrip: string) {
        loadingStops = true;

        try {
            stops = await fetchJSON(
                "/api/trip/" + encodeURIComponent(currentTrip) + "/stops",
            );
        } finally {
            loadingStops = false;
        }
    }

    $: if (trip && trip !== loadedTrip) {
        loadedTrip = trip;
        stops = [];
        loadStops(trip);
    }
</script>

<h1>Trip {trip}</h1>

<TripCard {trip} />

<h2>Stops</h2>

{#if loadingStops}
    <p class="muted">Loading stops...</p>
{:else if stops.length}
    <TripStopTable {stops} />
{:else}
    <p class="muted">No stops found.</p>
{/if}
