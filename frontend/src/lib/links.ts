export function stopURL(stopID?: string | null): string {
    if (!stopID) return "";
    return `/stop/${encodeURIComponent(stopID)}`;
}

export function routeURL(routeID?: string | null): string {
    if (!routeID) return "";
    return `/route/${encodeURIComponent(routeID)}`;
}

export function vehicleURL(
    dataOwner?: string | null,
    vehicleNumber?: string | number | null,
): string {
    if (!dataOwner || !vehicleNumber) return "";
    return `/vehicle/${encodeURIComponent(dataOwner)}/${encodeURIComponent(vehicleNumber)}`;
}

export function blockURL(
    dataOwner?: string | null,
    blockCode?: string | number | null,
): string {
    if (!dataOwner || !blockCode) return "";
    return `/block/${encodeURIComponent(dataOwner)}/${encodeURIComponent(blockCode)}`;
}
