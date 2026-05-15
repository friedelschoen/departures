<script lang="ts">
    import TripCard from "./TripCard.svelte";
    import type { Trip } from "../schema/Trip";
    import { vehicleURL, routeURL } from "../lib/links";

    export let trips: Trip[] = [];

    let openTrip: string | null = null;

    function hhmm(epoch?: number | null): string {
        if (!epoch) return "";
        return new Date(epoch * 1000).toLocaleTimeString("nl-NL", {
            hour: "2-digit",
            minute: "2-digit",
            hour12: false,
        });
    }

    function delayText(seconds?: number | null): string {
        const minutes = Math.trunc((seconds ?? 0) / 60);
        if (!minutes) return "";
        return `${minutes > 0 ? "+" : ""}${minutes} min`;
    }

    function lineStyle(t: Trip): string {
        let style = "text-align: center;";
        if (t.route_text_color) style += `color: #${t.route_text_color};`;
        if (t.route_color) style += `background-color: #${t.route_color};`;
        return style;
    }

    function toggleTrip(t: Trip) {
        openTrip = openTrip === t.trip_id ? null : t.trip_id;
    }
</script>

<table>
    <thead>
        <tr>
            <th></th>
            <th style="width:20px;"></th>
            <th>Destination</th>
            <th>Start</th>
            <th>End</th>
            <th>Vehicle</th>
            <th>Delay</th>
        </tr>
    </thead>

    <tbody>
        {#each trips as t}
            {@const open = openTrip === t.trip_id}

            <tr
                class:departure-row={true}
                class:open
                on:click={() => toggleTrip(t)}
            >
                <td class="toggle-cell">{open ? "▾" : "▸"}</td>

                <td style={lineStyle(t)}>
                    {#if t.route_id}
                        <a href={routeURL(t.route_id)}
                            >{t.route_short_name ?? ""}</a
                        >
                    {:else}
                        {t.route_short_name ?? ""}
                    {/if}
                </td>

                <td>
                    {t.trip_headsign ?? ""}
                    <div class="muted">
                        {#if t.start_stop_name || t.start_stop}
                            {t.start_stop_name ?? t.start_stop}
                        {/if}

                        {#if t.end_stop_name || t.end_stop}
                            → {t.end_stop_name ?? t.end_stop}
                        {/if}

                        {#if !t.start_stop_name && !t.start_stop}
                            trip {t.trip_id}
                        {/if}
                    </div>
                </td>

                <td>{hhmm(t.start_time ?? t.first_seen)}</td>
                <td>{hhmm(t.end_time ?? t.last_seen)}</td>
                <td>
                    {#if t.vehicle_number && t.data_owner_code}
                        <a
                            href={vehicleURL(
                                t.data_owner_code,
                                t.vehicle_number,
                            )}
                        >
                            {t.vehicle_number}
                        </a>
                    {:else}
                        {t.vehicle_number ?? ""}
                    {/if}
                </td>

                <td class:delay={(t.punctuality ?? 0) > 0}>
                    {delayText(t.punctuality)}
                </td>
            </tr>

            {#if open}
                <tr class="trip-card-row">
                    <td colspan="7">
                        <TripCard trip={t.trip_id} />
                    </td>
                </tr>
            {/if}
        {/each}
    </tbody>
</table>
