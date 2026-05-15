<script lang="ts">
    import TripTablePage from "./TripTablePage.svelte";

    export let data_owner: string;
    export let vehicle: string;

    $: endpoint =
        "/api/vehicle/" +
        encodeURIComponent(data_owner) +
        "/" +
        encodeURIComponent(vehicle);

    $: title = "Vehicle " + vehicle;
</script>

<TripTablePage
    {title}
    {endpoint}
    emptyText="No trips found for this vehicle."
/>
