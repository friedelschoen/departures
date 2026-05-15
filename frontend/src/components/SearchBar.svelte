<script lang="ts">
    import { navigate } from "svelte-routing";
    import type { Query } from "../schema/Query";

    let q = "";
    let results: Query[] = [];
    let loading = false;
    let selected = -1;
    let timer: number | null = null;

    function routeStyle(r: Query): string {
        let style = "";
        if (r.route_text_color) style += `color: #${r.route_text_color};`;
        if (r.route_color) style += `background-color: #${r.route_color};`;
        return style;
    }

    function resultURL(r: Query): string {
        switch (r.type) {
            case "stop":
                return `/stop/${encodeURIComponent(r.stop_id ?? "")}`;
            case "route":
                return `/route/${encodeURIComponent(r.route_id ?? "")}`;
            case "vehicle":
                return `/vehicle/${encodeURIComponent(r.data_owner_code ?? "")}/${encodeURIComponent(r.vehicle_number ?? "")}`;
            case "block":
                return `/block/${encodeURIComponent(r.data_owner_code ?? "")}/${encodeURIComponent(r.block_code ?? "")}`;
            default:
                return "/";
        }
    }

    async function search() {
        const query = q.trim();

        if (query.length == 0) {
            results = [];
            selected = -1;
            return;
        }

        loading = true;

        try {
            const res = await fetch(
                `/api/search?q=${encodeURIComponent(query)}`,
            );
            const json = await res.json();

            if (!res.ok) throw new Error(json?.error ?? res.statusText);

            results = json.result ?? [];
            selected = results.length ? 0 : -1;
        } finally {
            loading = false;
        }
    }

    function onInput() {
        if (timer != null) clearTimeout(timer);

        timer = window.setTimeout(search, 200);
    }

    function open(r: Query) {
        q = "";
        results = [];
        selected = -1;
        navigate(resultURL(r));
    }

    function onKeydown(ev: KeyboardEvent) {
        if (!results.length) return;

        switch (ev.key) {
            case "ArrowDown":
                ev.preventDefault();
                selected = Math.min(selected + 1, results.length - 1);
                break;
            case "ArrowUp":
                ev.preventDefault();
                selected = Math.max(selected - 1, 0);
                break;
            case "Enter":
                ev.preventDefault();
                if (selected >= 0) open(results[selected]);
                break;
            case "Escape":
                results = [];
                selected = -1;
                break;
        }
    }

    function typeLabel(type: string): string {
        switch (type) {
            case "stop":
                return "Stop";
            case "route":
                return "Route";
            case "vehicle":
                return "Vehicle";
            case "block":
                return "Block";
            default:
                return type;
        }
    }
</script>

<div class="searchbar">
    <input
        type="text"
        bind:value={q}
        on:input={onInput}
        on:keydown={onKeydown}
        placeholder="Search stop, route, vehicle or block..."
        autocomplete="off"
    />

    {#if results.length || loading}
        <div class="search-results">
            {#if loading}
                <div class="search-loading">Searching...</div>
            {/if}

            {#each results as r, i}
                <button
                    type="button"
                    class:selected={i === selected}
                    on:click={() => open(r)}
                >
                    <span class="search-type">{typeLabel(r.type)}</span>

                    {#if r.type === "route"}
                        <span class="route-pill" style={routeStyle(r)}>
                            {r.route_short_name ?? r.label}
                        </span>
                    {/if}

                    <span class="search-main">{r.label}</span>

                    {#if r.subtitle}
                        <span class="search-subtitle">{r.subtitle}</span>
                    {/if}
                </button>
            {/each}
        </div>
    {/if}
</div>
