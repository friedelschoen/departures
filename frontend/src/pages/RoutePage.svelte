<script lang="ts">
    import TripTablePage from "./TripTablePage.svelte";

    export let route: string;

    $: endpoint = "/api/route/" + encodeURIComponent(route) + "/trips";
    $: title = "Route " + route;
</script>

<TripTablePage {title} {endpoint} emptyText="No trips found for this route." />
