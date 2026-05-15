<script lang="ts">
    import TripTable from "../components/TripTable.svelte";
    import type { Trip } from "../schema/Trip";

    export let title: string;
    export let endpoint: string;
    export let emptyText = "No trips found.";

    let trips: Trip[] = [];
    let loading = false;

    async function fetchJSON(url: URL | RequestInfo) {
        const res = await fetch(url);
        const json = await res.json();
        if (!res.ok) throw new Error(json?.error ?? res.statusText);
        return json.result;
    }

    async function loadTrips() {
        if (!endpoint) return;

        loading = true;
        try {
            trips = await fetchJSON(endpoint);
        } finally {
            loading = false;
        }
    }

    let loadedEndpoint = "";

    $: if (endpoint && endpoint !== loadedEndpoint) {
        loadedEndpoint = endpoint;
        loadTrips();
    }
</script>

<h1>{title}</h1>

{#if loading}
    <p class="muted">Loading trips...</p>
{:else if trips.length}
    <TripTable {trips} />
{:else}
    <p class="muted">{emptyText}</p>
{/if}
