<script lang="ts">
    import TripTablePage from "./TripTablePage.svelte";

    export let data_owner: string;
    export let block: string;

    $: endpoint =
        "/api/block/" +
        encodeURIComponent(data_owner) +
        "/" +
        encodeURIComponent(block);

    $: title = "Block " + block;
</script>

<TripTablePage {title} {endpoint} emptyText="No trips found for this block." />
