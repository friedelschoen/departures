function esc(s) {
	return String(s ?? "").replace(/[&<>"']/g, c => ({
		"&": "&amp;",
		"<": "&lt;",
		">": "&gt;",
		'"': "&quot;",
		"'": "&#39;"
	}[c]));
}

function insertNowRow(rows) {
	const tr = document.createElement("tr");
	tr.className = "now-row";
	tr.innerHTML = `<td colspan="7"><div class="now-line"><span>nu</span></div></td>`;
	rows.appendChild(tr);
}

function hhmm(d) {
	return d.toLocaleTimeString("nl-NL", {
		timeZone: "Europe/Amsterdam",
		hour: "2-digit",
		minute: "2-digit",
		hour12: false
	});
}

function lineStyle(d) {
	let style = "text-align: center;";
	if (d.fg_color) style += `color: #${esc(d.fg_color)};`;
	if (d.bg_color) style += `background-color: #${esc(d.bg_color)};`;
	return style;
}

function delayText(minutes) {
	if (!minutes) return "";
	return `${minutes > 0 ? "+" : ""}${minutes} min`;
}

function statusText(status, stop) {
    switch (status) {
        case "DELAY":
            return " is delayed";
        case "INIT":
            return ` is at the starting point <b>${stop}</b>`;
        case "ARRIVAL":
            return ` is arriving at <b>${stop}</b>`;
        case "ONSTOP":
            return ` is stopping at <b>${stop}</b>`;
        case "DEPARTURE":
            return ` passed <b>${stop}</b>`;
        case "ONROUTE":
            return ` near <b>${stop}</b>`;
        case "OFFROUTE":
            return ` is off route near <b>${stop}</b>`;
        case "END":
            return ` has finished at <b>${stop}</b>`;
        default:
            return "";
    }
}

function vehicleContextText(v) {
	if (!v || !v.status) return "";

	const delay = v.delay_minutes
		? ` (${delayText(v.delay_minutes)})`
		: "";
    
    const lastseen = (Date.now() / 1000) - v.last_seen;
	const lastseenText = !v.last_seen ? ''
        : lastseen < 30 ? " &mdash; just now"
        : lastseen < 60 ? ` &mdash; ${Math.round(lastseen)}s ago`
        : ` &mdash; ${Math.round(lastseen / 60)}min ago`;
    
    return `
		<span class="context-label">currently</span>
		<b>${esc(v.line)} ${esc(v.headsign ?? "")}</b>
		&mdash; ${statusText(v.status, esc(v.stop_name))}</b>
		${delay}${lastseenText}
	`;
}

function insertContextRow(rows, text) {
	if (!text) return;

	const tr = document.createElement("tr");
	tr.className = "context-row next";
	tr.innerHTML = `<td colspan="7">${text}</td>`;
	rows.appendChild(tr);
}

function vehicleLink(d) {
	if (!d.id || !d.latitude || !d.longitude) return "";

	const id = esc(d.id);
	const lat = Number(d.latitude).toFixed(5);
	const lon = Number(d.longitude).toFixed(5);

	return `<a target="_blank" href="https://www.openstreetmap.org/?mlat=${lat}&mlon=${lon}#map=16/${lat}/${lon}">${id}</a>`;
}

function renderDepartureRow(d, markDelay) {
	const tr = document.createElement("tr");
	tr.className = "departure-row";

	if (d.vehicle && d.vehicle.status) tr.classList.add("has-next");
	if (d.cancelled) tr.classList.add("cancelled");
	if (d.terminal) tr.classList.add("terminal");
    if (d.warning) tr.classList.add("alert-high");

	const scheduleTime = new Date(d.scheduled_time * 1000);
	const realtimeTime = new Date(d.realtime_time * 1000);
	const lastseen = (Date.now() / 1000) - d.last_seen;

	const time = d.scheduled_time === d.realtime_time
		? `<b>${hhmm(scheduleTime)}</b>`
		: `<s>${esc(hhmm(scheduleTime))}</s> <b>${esc(hhmm(realtimeTime))}</b>`;

    const lastseenText = !d.last_seen ? "" 
        : lastseen < 30 ? "Now"
        : lastseen < 60 ? `${Math.round(lastseen)}s ago`
        : `${Math.round(lastseen / 60)}min ago`;

	let delay = d.delay_minutes;

    if (markDelay) delay = d.delay_minutes >= 0
		? (d.delay_minutes !== 0 ? "+" + d.delay_minutes : "0")
		: d.delay_minutes;

	const vehicle = vehicleLink(d);

	tr.innerHTML = `
		<td style="text-align: center;">${esc(d.platform)}</td>
		<td style="${lineStyle(d)}">${esc(d.line)}</td>
		<td>${esc(d.headsign)}</td>
        <!--<td>${d.blockcode ? esc(d.blockcode) : ""}</td>-->
		<td>${time}</td>
		<td class="${d.delay_minutes > 0 && markDelay ? "delay" : ""}">
			${d.cancelled ? "cancelled" : (vehicle || d.delay_minutes !== 0 ? delay + " min" : "")}
		</td>
		<td>${vehicle}</td>
		<td>${lastseenText}</td>
	`;

	return tr;
}

function insertDateRow(rows, date) {
	const tr = document.createElement("tr");
	tr.innerHTML = `<td colspan="7" class="date-row">${esc(date)}</td>`;
	rows.appendChild(tr);
}

async function loadStopInfo() {
	const params = new URL(document.location.toString()).searchParams;
	const stop = params.get("stop");

	const res = await fetch("/api/stop/" + encodeURIComponent(stop));
	const stopinfo = await res.json();

	document.getElementById("stopname").innerHTML = `Departures for <i>${esc(stopinfo.result.stop_name)}</i>`;
}

async function loadDepartures(element, endpoint, insertNow = true, markDelay = true) {
	const params = new URL(document.location.toString()).searchParams;
	const stop = params.get("stop");
	const showTerminal = params.get("show-terminal");

    const res = await fetch(`/api/${endpoint}/` + encodeURIComponent(stop));
	const result = await res.json();
    const deps = result.result ?? [];

	const rows = document.getElementById(element);
	rows.innerHTML = "";

	let prevDate = "";
	let nowInserted = !insertNow;
	const now = new Date();

	for (const d of deps) {
		if (!showTerminal && d.terminal) continue;

		const scheduleTime = new Date(d.scheduled_time * 1000);
		const date = scheduleTime.toLocaleDateString("nl-NL", {
			timeZone: "Europe/Amsterdam",
			day: "2-digit",
			month: "2-digit",
			year: "numeric"
		});

		if (date !== prevDate) {
			prevDate = date;
			insertDateRow(rows, date);
		}

		if (!nowInserted && scheduleTime > now) {
			insertNowRow(rows);
			nowInserted = true;
		}

		rows.appendChild(renderDepartureRow(d, markDelay));
		insertContextRow(rows, vehicleContextText(d.vehicle));
	}

	if (!nowInserted) {
		insertNowRow(rows);
	}
}

const stopInput = document.getElementById("stop");
const stopResults = document.getElementById("stop-results");

stopInput.addEventListener("input", async () => {
	const q = stopInput.value.trim();

	if (q.length < 2) {
		stopResults.innerHTML = "";
		return;
	}

	const res = await fetch("/api/stop_query?q=" + encodeURIComponent(q));
	const stops = await res.json();

	stopResults.innerHTML = "";

	for (const s of stops.result) {
		const div = document.createElement("div");
		div.style.padding = "4px";
		div.style.cursor = "pointer";

		div.textContent = `${s.name} (${s.id})`;

		div.onclick = () => {
            stopInput.value = s.id;
		};

		stopResults.appendChild(div);
	}
});

loadStopInfo();
loadDepartures("rows", "departures");
loadDepartures("buffer", "buffer", false, false);
setInterval(() => {
    loadDepartures("rows", "departures");
    loadDepartures("buffer", "buffer", false, false);
}, 15000);

