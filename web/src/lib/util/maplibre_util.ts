import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
import { haversineDistance } from "$lib/models/gpx/utils";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";
import { theme } from "$lib/stores/theme_store";
import M from "maplibre-gl";
import { _ } from "svelte-i18n";
import { get } from "svelte/store";
import { handleFromRecordWithIRI } from "./activitypub_util";
import { getFileURL } from "./file_util";
import { formatDistance, formatElevation, formatTimeHHMM } from "./format_util";
import { icons } from "./icon_util";

export class FontawesomeMarker extends M.Marker {
    constructor(options: { icon: string, fontSize?: string, width?: number, backgroundColor?: string, fontColor?: string, style?: string, id?: string }, markerOptions?: M.MarkerOptions) {
        const element = document.createElement('div')
        element.className = `cursor-pointer flex items-center justify-center w-${options.width ?? 7} aspect-square ${options.backgroundColor ?? "bg-gray-500"} rounded-full text-${options.fontSize ?? "normal"} ${options.style ?? ""}`
        element.id = options.id ?? "";
        super({ element: element, ...markerOptions });

        const {
            icon,
        } = options

        const iconElementString = `<i class="text-${options.fontColor ?? "white"} ${icon}"></i>`
        this._element.insertAdjacentHTML('beforeend', iconElementString)
    }
}

export function createMarkerFromWaypoint(waypoint: Waypoint, onDragEnd?: (marker: M.Marker, wpId?: string) => void): FontawesomeMarker {
    const marker = new FontawesomeMarker({
        id: waypoint.id,
        icon: `fa fa-${waypoint.icon}`,
    }, {
        draggable: onDragEnd !== undefined,
        color: "#6b7280"

    })

    const content = document.createElement("div");
    content.className = "p-2"

    const spanElement = document.createElement("span");
    const iconElement = document.createElement("i");
    const iconName = waypoint.icon && icons.includes(waypoint.icon) ? waypoint.icon : "circle";
    iconElement.classList.add("fa", `fa-${iconName}`)
    spanElement.appendChild(iconElement);

    const nameElement = document.createElement("b");
    nameElement.textContent = waypoint.name ?? "-";
    if (waypoint.name?.length) {
        nameElement.classList.add("ml-2")
    }
    spanElement.appendChild(nameElement);
    content.appendChild(spanElement);

    if (waypoint.description && waypoint.description.length > 0) {
        const descriptionElement = document.createElement("p");
        descriptionElement.textContent = waypoint.description;
        content.appendChild(descriptionElement);
    }


    const popup = new M.Popup({ offset: 25 }).setDOMContent(
        content
    );
    marker
        .setLngLat([waypoint.lon, waypoint.lat])
        .setPopup(popup)

    if (onDragEnd) {
        marker.on("dragend", () => onDragEnd(marker, waypoint.id,));
    }

    return marker;
}

export function createAnchorMarker(lat: number, lon: number, index: number,
    onDeleteClick: () => void, onLoopClick: () => void,
    onDragStart: (event: Event) => void, onDragEnd: (event: Event) => void): FontawesomeMarker {

    const anchorElement = document.createElement("span")
    anchorElement.className = "route-anchor cursor-pointer rounded-full w-6 h-6 border border-black text-center bg-primary text-white"
    anchorElement.textContent = "" + index
    const marker = new M.Marker(
        {
            draggable: true,
            element: anchorElement
        }
    );
    marker.setLngLat([lon, lat]);
    const popup = new M.Popup()

    const popupContent = document.createElement("div");
    popupContent.className = "py-3 pl-3"
    const anchorH = document.createElement("h5")
    anchorH.classList.add("text-base", "font-medium");
    anchorH.textContent = get(_)("route-point") + " #" + index;

    const deleteButton = document.createElement("button");
    deleteButton.className = "btn-secondary w-full mt-2 text-sm";
    const deleteButtonIcon = document.createElement("i")
    deleteButtonIcon.classList.add("fa", "fa-trash", "mr-2")
    deleteButton.appendChild(deleteButtonIcon)
    const deleteButtonText = document.createElement("span")
    deleteButtonText.textContent = get(_)("delete")
    deleteButton.appendChild(deleteButtonIcon)
    deleteButton.appendChild(deleteButtonText)
    deleteButton.addEventListener("click", onDeleteClick)

    const loopButton = document.createElement("button");
    loopButton.className = "btn-secondary w-full mt-2 text-sm block";
    const loopButtonIcon = document.createElement("i")
    loopButtonIcon.classList.add("fa", "fa-person-walking-arrow-loop-left", "mr-2")
    loopButton.appendChild(loopButtonIcon)
    const loopButtonText = document.createElement("span")
    loopButtonText.textContent = get(_)("loop")
    loopButton.appendChild(loopButtonIcon)
    loopButton.appendChild(loopButtonText)
    loopButton.addEventListener("click", onLoopClick)

    popupContent.appendChild(anchorH)
    popupContent.appendChild(deleteButton)
    popupContent.appendChild(loopButton)
    popup.setDOMContent(popupContent)
    marker.setPopup(popup);

    marker.on("dragstart", onDragStart);
    marker.on("dragend", onDragEnd);
    marker.getElement().addEventListener("click", (e) => {
        e.preventDefault()
        e.stopPropagation();
        marker.togglePopup();
    })

    return marker
}

export function createEditTrailMapPopup(lnglat: M.LngLat, onCreateWaypointClick: () => void) {
    const popup = new M.Popup({ closeOnClick: false })
        .setLngLat(lnglat)

    const popupContent = document.createElement("div");
    popupContent.className = "pt-1 pb-3 pl-3"
    const popupH = document.createElement("h5")
    popupH.classList.add("text-base", "font-medium", "mb-2");
    popupH.textContent = `${lnglat.lat.toFixed(4)}, ${lnglat.lng.toFixed(4)}`;

    const createWaypointButton = document.createElement("button");
    createWaypointButton.className = "btn-secondary w-full mt-2 text-sm";
    const deleteButtonIcon = document.createElement("i")
    deleteButtonIcon.classList.add("fa", "fa-location-dot", "mr-2")
    createWaypointButton.appendChild(deleteButtonIcon)
    const createWaypointButtonText = document.createElement("span")
    createWaypointButtonText.textContent = get(_)("create-waypoint")
    createWaypointButton.appendChild(deleteButtonIcon)
    createWaypointButton.appendChild(createWaypointButtonText)
    createWaypointButton.addEventListener("click", onCreateWaypointClick)

    popupContent.appendChild(popupH)
    popupContent.appendChild(createWaypointButton)

    popup.setDOMContent(popupContent)

    return popup
}

export function createPopupFromTrail(trail: Trail) {
    const thumbnail = trail.photos.length
        ? getFileURL(trail, trail.photos.at(trail.thumbnail ?? 0) ?? trail.photos[0])
        : get(theme) === "light"
            ? emptyStateTrailLight
            : emptyStateTrailDark;
    const popup = new M.Popup({ maxWidth: "420px" });
    if (!trail.expand?.author) {
        return popup
    }
    // Create a container element for the popup content
    const linkElement = document.createElement("a");
    linkElement.href = `/map/trail/${handleFromRecordWithIRI(trail)}/${trail.id}`; // Set href safely
    linkElement.setAttribute("data-sveltekit-preload-data", "off");

    // Create a list item element
    const listItem = document.createElement("li");
    listItem.className = "flex gap-4 cursor-pointer text-content max-w-72";

    // Create the image container
    const imageContainer = document.createElement("div");
    imageContainer.className = "shrink-0";

    // Create the image element
    const img = document.createElement("img");
    img.className = "h-full w-20 object-cover";
    img.src = thumbnail; // Set image source safely
    img.alt = ""; // Always include a safe alt attribute
    imageContainer.appendChild(img);

    // Create the text container
    const textContainer = document.createElement("div");
    textContainer.className = "py-2"

    // Add trail name
    const trailName = document.createElement("h4");
    trailName.className = "font-semibold text-lg line-clamp-1";
    trailName.textContent = trail.name; // Set trail name safely
    textContainer.appendChild(trailName);

    // Add location and difficulty, if available
    if (trail.location || trail.difficulty) {
        const detailsContainer = document.createElement("div");
        detailsContainer.className = "flex gap-x-4";

        if (trail.location) {
            const locationElement = document.createElement("h5");
            locationElement.innerHTML = `<i class="fa fa-location-dot mr-2"></i>`; // Safe static icon
            locationElement.appendChild(document.createTextNode(trail.location)); // Safely append location text
            detailsContainer.appendChild(locationElement);
        }

        const difficultyElement = document.createElement("h5");
        difficultyElement.innerHTML = `<i class="fa fa-gauge mr-2"></i>`; // Safe static icon
        difficultyElement.appendChild(document.createTextNode(get(_)(trail.difficulty as string))); // Safely append difficulty
        detailsContainer.appendChild(difficultyElement);

        textContainer.appendChild(detailsContainer);
    }

    // Create the grid container for additional stats
    const statsContainer = document.createElement("div");
    statsContainer.className =
        "grid grid-cols-2 mt-2 gap-x-4 text-gray-500 flex-wrap";

    const stats = [
        { icon: "fa-left-right", value: formatDistance(trail.distance) },
        { icon: "fa-clock", value: formatTimeHHMM(trail.duration) },
        { icon: "fa-arrow-trend-up", value: formatElevation(trail.elevation_gain) },
        { icon: "fa-arrow-trend-down", value: formatElevation(trail.elevation_loss) },
    ];

    // Loop through stats and add them
    stats.forEach(({ icon, value }) => {
        const statElement = document.createElement("span");
        statElement.className = "shrink-0";
        statElement.innerHTML = `<i class="fa ${icon} mr-2"></i>`; // Safe static icon
        statElement.appendChild(document.createTextNode(value)); // Safely append stat value
        statsContainer.appendChild(statElement);
    });

    textContainer.appendChild(statsContainer);

    // Assemble the popup
    listItem.appendChild(imageContainer);
    listItem.appendChild(textContainer);
    linkElement.appendChild(listItem);

    // Safely set the content using setDOMContent
    popup.setDOMContent(linkElement);

    return popup;
}

export function createOverpassPopup(feature: GeoJSON.Feature, coordinates: GeoJSON.Position) {
    const tags: Record<string, string> = JSON.parse(feature.properties?.tags);
    const name = tags.name ?? get(_)(feature.properties?.query) ?? "?"

    const popupContainer = document.createElement("div");
    popupContainer.className = "p-4"

    const popupHeading = document.createElement("h1");
    popupHeading.className = "font-medium text-lg"
    popupHeading.textContent = name;

    const coordinateSubtitle = document.createElement("p")
    coordinateSubtitle.className = "text-gray-500"
    coordinateSubtitle.textContent = `${coordinates[0].toFixed(6)}, ${coordinates[1].toFixed(6)}`

    popupContainer.appendChild(popupHeading)
    popupContainer.appendChild(coordinateSubtitle)

    const tagsGrid = document.createElement("div")
    tagsGrid.className = "grid grid-cols-2 gap-x-4 mt-4"

    Object.entries(tags).forEach((([k, v]) => {
        if (k == "name") return;
        const kSpan = document.createElement("span")
        kSpan.className = "font-mono"
        kSpan.textContent = k;
        const vSpan = document.createElement("span")
        vSpan.textContent = v;

        tagsGrid.appendChild(kSpan);
        tagsGrid.appendChild(vSpan);
    }));

    popupContainer.appendChild(tagsGrid)

    return popupContainer;
}

export function calculatePixelPerMeter(map: M.Map, meters: number) {
    const y = map.getCanvas().getBoundingClientRect().y;
    const x = map.getCanvas().getBoundingClientRect().x;
    const maxMeters = map.unproject([0, y]).distanceTo(map.unproject([x, y]));
    const pixelPerMeter = x / maxMeters;

    return pixelPerMeter * meters
}

export function calculateScaleFactor(map: M.Map) {
    function _pxTOmm() {
        let heightRef = document.createElement('div');
        heightRef.style.height = '1mm';
        heightRef.style.position = "absolute";
        heightRef.id = 'heightRef';
        document.body.appendChild(heightRef);

        const pxPermm = heightRef.getBoundingClientRect().height;

        document.body.removeChild(heightRef);

        return function pxTOmm(px: number) {
            return px / pxPermm;
        }
    }
    var centerOfMap = map.getCanvas().getBoundingClientRect().y / 2;

    const p1 = map.unproject([0, centerOfMap]);
    const p2 = map.unproject([100, centerOfMap]);
    var realWorldMetersPer100Pixels = haversineDistance(
        p1.lat, p1.lng, p2.lat, p2.lng
    );

    const screenMetersPer100Pixels = _pxTOmm()(100) / 1000;

    const scaleFactor = realWorldMetersPer100Pixels / screenMetersPer100Pixels

    return scaleFactor
}