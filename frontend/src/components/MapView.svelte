<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import L from "leaflet";
    import type { Vehicle } from "../schema/Vehicle";
    import type { Trip } from "../schema/Trip";
    import type { TripShape } from "../schema/TripShape";

    export let tripID = "";
    export let vehicle: Vehicle | Trip | null = null;
    export let focus: "line" | "vehicle" = "line";

    let el: HTMLDivElement;
    let map: L.Map | null = null;
    let routeLayer: L.Polyline | null = null;
    let vehicleMarker: L.Marker | L.CircleMarker | null = null;

    function vehicleLatLon() {
        if (!vehicle?.lat || !vehicle?.lon) return null;
        return [vehicle.lat, vehicle.lon] as [number, number];
    }

    async function loadShape() {
        if (!tripID || !map) return;

        const res = await fetch(
            `/api/trip/${encodeURIComponent(tripID)}/shape`,
        );
        const json = await res.json();
        const shape: TripShape[] = json?.result;

        const latlngs: L.LatLngExpression[] = shape.map((p: TripShape) => [
            p.lat,
            p.lon,
        ]);

        if (routeLayer) {
            routeLayer.remove();
            routeLayer = null;
        }

        if (latlngs.length >= 2) {
            routeLayer = L.polyline(latlngs, {
                weight: 4,
            }).addTo(map);

            map.fitBounds(routeLayer.getBounds(), {
                padding: [20, 20],
            });
        }
        updateVehicleMarker();
    }

    function updateVehicleMarker() {
        if (!map) return;

        const pos = vehicleLatLon();

        if (!pos) {
            if (vehicleMarker) {
                vehicleMarker.remove();
                vehicleMarker = null;
            }
            return;
        }
        if (!vehicleMarker) {
            vehicleMarker = L.circleMarker(pos, {
                color: "#333333",
                fillColor: "#ffc40c",
                fillOpacity: 0.5,
            }).addTo(map);
        } else {
            vehicleMarker.setLatLng(pos);
        }
        if (focus == "vehicle") map.setView(pos, 18);
    }

    onMount(() => {
        map = L.map(el);

        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            maxZoom: 19,
            attribution: "&copy; OpenStreetMap contributors",
        }).addTo(map);

        if (tripID || vehicle) loadShape();
    });

    onDestroy(() => {
        map?.remove();
        map = null;
    });

    let loadedTripID = "";

    $: if (map && tripID && tripID !== loadedTripID) {
        loadedTripID = tripID;
        loadShape();
    }
</script>

<div bind:this={el} class="leaflet-map"></div>
