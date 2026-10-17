import type { Waypoint } from "$lib/models/waypoint";
import { getFileURL, isVideoURL } from "./file_util";

export type WaypointPopupMedia = {
    url: string;
    video: boolean;
    revoke?: () => void;
};

export const WAYPOINT_FOCUS_EVENT = "wanderer:focus-waypoint";
export const WAYPOINT_POPUP_MEDIA_LIMIT = 3;

export type WaypointFocusDetail = {
    waypointId: string;
    lat: number;
    lon: number;
    source: "map" | "profile";
};

export type WaypointFocusTarget = {
    id?: string;
    lat: number;
    lon: number;
};

let activeElevationWaypointId: string | undefined;

export function waypointAnchorId(waypointId?: string) {
    return waypointId ? `waypoint-${waypointId}` : undefined;
}

export function waypointHasPopupMedia(waypoint: Waypoint) {
    return (
        (waypoint.photos?.length ?? 0) > 0 ||
        (waypoint._photos?.length ?? 0) > 0
    );
}

export function getWaypointPopupMedia(
    waypoint: Waypoint,
    thumb?: string,
): WaypointPopupMedia[] {
    const media: WaypointPopupMedia[] = [];

    for (const photo of waypoint.photos ?? []) {
        const url = getFileURL(waypoint, photo, thumb);
        if (url) {
            media.push({
                url,
                video: isVideoURL(url),
            });
        }
    }

    for (const file of waypoint._photos ?? []) {
        const url = URL.createObjectURL(file);
        media.push({
            url,
            video: isVideoURL(file.name) || file.type.startsWith("video/"),
            revoke: () => URL.revokeObjectURL(url),
        });
    }

    return media;
}

export function revokeWaypointPopupMedia(media: WaypointPopupMedia[]) {
    for (const item of media) {
        item.revoke?.();
        item.revoke = undefined;
    }
}

export function scrollToWaypointAnchor(waypointId?: string) {
    const id = waypointAnchorId(waypointId);
    if (!id || typeof document === "undefined") {
        return null;
    }

    const element = document.getElementById(id);
    element?.scrollIntoView({ behavior: "smooth", block: "center" });
    return element ?? null;
}

export function highlightElevationWaypoint(waypointId?: string) {
    activeElevationWaypointId = waypointId;
    applyElevationWaypointHighlight();
}

export function applyElevationWaypointHighlight() {
    if (typeof document === "undefined") {
        return;
    }

    document.querySelectorAll(".wp-marker-active").forEach((element) => {
        element.classList.remove("wp-marker-active");
    });

    if (!activeElevationWaypointId) {
        return;
    }

    const selector = `.wp-marker[data-waypoint-id="${activeElevationWaypointId}"]`;
    document.querySelector(selector)?.classList.add("wp-marker-active");
}

export function focusWaypoint(
    waypoint?: WaypointFocusTarget | null,
    source: WaypointFocusDetail["source"] = "map",
) {
    if (!waypoint?.id) {
        return;
    }

    scrollToWaypointAnchor(waypoint.id);
    highlightElevationWaypoint(waypoint.id);

    if (typeof document === "undefined") {
        return;
    }

    document.dispatchEvent(
        new CustomEvent<WaypointFocusDetail>(WAYPOINT_FOCUS_EVENT, {
            detail: {
                waypointId: waypoint.id,
                lat: waypoint.lat,
                lon: waypoint.lon,
                source,
            },
        }),
    );
}
