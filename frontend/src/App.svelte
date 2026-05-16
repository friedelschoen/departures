<script>
    import { Router, Route } from "svelte-routing";
    import SearchBar from "./components/SearchBar.svelte";
    import StopPage from "./pages/StopPage.svelte";
    import RoutePage from "./pages/RoutePage.svelte";
    import BlockPage from "./pages/BlockPage.svelte";
    import VehiclePage from "./pages/VehiclePage.svelte";
    import TripPage from "./pages/TripPage.svelte";

    export let url = "";
</script>

<SearchBar />

<Router {url}>
    <Route path="/stop/:stop" let:params>
        <StopPage stop={params.stop} />
    </Route>
    <Route path="/block/:data_owner/:block" let:params>
        <BlockPage data_owner={params.data_owner} block={params.block} />
    </Route>
    <Route path="/vehicle/:data_owner/:vehicle" let:params>
        <VehiclePage data_owner={params.data_owner} vehicle={params.vehicle} />
    </Route>
    <Route path="/route/:route" let:params>
        <RoutePage route={params.route} />
    </Route>
    <Route path="/trip/:trip" let:params>
        <TripPage trip={params.trip} />
    </Route>
    <Route path="/">
        <StopPage stop={"stoparea:449933"} />
    </Route>
    <Route>404 not found</Route>
</Router>

<footer class="footer">
    &copy; 2026 Friedel Schön &mdash; Source hosted at
    <a href="https://github.com/friedelschoen/departures" target="_blank">
        GitHub
    </a>
</footer>
