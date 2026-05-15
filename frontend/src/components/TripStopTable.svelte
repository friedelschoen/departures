<script lang="ts">
    import type { TripStop } from "../schema/TripStop";
    import { stopURL } from "../lib/links";

    export let stops: TripStop[] = [];

    function hhmm(d: Date): string {
        return d.toLocaleTimeString("nl-NL", {
            hour: "2-digit",
            minute: "2-digit",
            hour12: false,
        });
    }

    function delayText(minutes: number): string {
        if (!minutes) return "0 min";
        return `${minutes > 0 ? "+" : ""}${minutes} min`;
    }

    function delayMin(p: number): number {
        return Math.trunc((p ?? 0) / 60);
    }

    function filterDuplicate(stops: TripStop[]): TripStop[] {
        const out: TripStop[] = [];

        for (let i = 0; i < stops.length; i++) {
            const cur = stops[i];
            const next = stops[i + 1];

            if (
                next &&
                cur.stop_id === next.stop_id &&
                Math.abs(cur.scheduled_time - next.scheduled_time) <= 60 &&
                Math.abs(cur.punctuality - next.punctuality) <= 60
            ) {
                out.push({
                    ...cur,
                    mode: "",
                });

                i++;
                continue;
            }

            out.push(cur);
        }

        return out;
    }
</script>

<table class="trip-stops">
    <thead>
        <tr>
            <th></th>
            <th>Stop</th>
            <th>Time</th>
            <th>Delay</th>
        </tr>
    </thead>

    <tbody>
        {#each filterDuplicate(stops) as st}
            {@const delay = delayMin(st.punctuality)}

            <tr
                class:passed-stop={st.vehicle_status === "passed"}
                class:active-stop={st.vehicle_status === "active"}
                class:alert-high={st.vehicle_status === "offroute"}
            >
                <td class="event-mode">
                    {st.mode == "" ? "" : st.mode === "arrival" ? "A" : "D"}
                </td>

                <td>
                    <a href={stopURL(st.stop_id)}>{st.stop_name}</a>

                    {#if st.platform_code}
                        <span class="muted">platform {st.platform_code}</span>
                    {/if}
                </td>

                <td>{hhmm(new Date(st.scheduled_time * 1000))}</td>

                <td class="trip-delay">
                    {#if st.status}
                        {delayText(delay)}{st.status === "ONROUTE" ? "?" : ""}
                    {/if}
                </td>
            </tr>
        {/each}
    </tbody>
</table>
