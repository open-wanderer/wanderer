import { type ControlPosition, type IControl } from "maplibre-gl";
import * as M from "maplibre-gl";
import { ElevationProfile, type ElevationProfileOptions } from "./elevationprofile";
// @ts-ignore
import type { GeoJsonObject, Position } from "geojson";
import type { Waypoint } from "$lib/models/waypoint";

/**
 * Elevation profile control options
 */
export type ElevationProfileControlOptions = ElevationProfileOptions & {
    /**
     * If `true`, the elevation profile control will be visible as soon as it's ready.
     * If `false`, a click on the control button (or a programmatic call to `.showProfile()`)
     * will be neccesary to show the profile.
     *
     * Default: `false`
     */
    visible?: boolean;
    /**
     * Size of the profile as a CSS rule.
     * This `size` will be the `width` if the `.position` is "left" or "right",
     * and will be the `height` if the `.position` is "top" or "bottom".
     *
     * Default: `"30%"`
     */
    size?: string;
    /**
     * Position of the elevation profile chart when shown.
     *
     * Default: `"botton"`
     */
    position?: "top" | "left" | "right" | "bottom";
    /**
     * Show the control button. If can be handy to hide it, especially if the profile is displayed
     * in a custom container and that its visiblity is managed by logic external to this control.
     *
     * Default: `true`
     */
    showButton?: boolean;
    /**
     * A CSS class to add to the container. This is especially relevant when the options `.container` is not provided.
     * Important: if provided, no styling is added by this control and even placement will have to be managed by external CSS.
     *
     * Default: `""`
     */
    containerClass?: string;

    /**
     * DIV element to contain the control.
     * Important: if provided, no styling is added by this control.
     * Default: automatically created inside the map container
     */
    container?: string | HTMLDivElement;
    /**
     * Called when the profile visibility is toggled through the control.
     */
    onToggle?: (visible: boolean) => void;
};

export class ElevationProfileControl implements IControl {
    private map?: M.Map;
    private buttonContainer?: HTMLDivElement;
    private toggleButton?: HTMLButtonElement;
    public isProfileShown = false;
    private iconSpan?: HTMLSpanElement;

    private profileContainer?: HTMLDivElement;
    private settings: ElevationProfileControlOptions;
    private data: GeoJsonObject | null = null;
    private elevationProfileChart?: ElevationProfile;

    constructor(options: ElevationProfileControlOptions = {}) {
        if (typeof window === "undefined")
            throw new Error("This pluggin must be mounted client-side");
        this.settings = { ...options };
    }

    toggleTheme(options: ElevationProfileControlOptions) {
        this.settings = { ...this.settings, ...options };
        this.elevationProfileChart?.toggleTheme(this.settings)
    }

    getContainer(): HTMLDivElement | undefined {
        return this.profileContainer;
    }

    onAdd(map: M.Map): HTMLElement {
        this.map = map;

        this.buttonContainer = document.createElement("div");

        if (this.settings.showButton === false) {
            this.buttonContainer.style.setProperty("display", "none");
        }

        this.buttonContainer.classList.add(
            "maplibregl-ctrl",
            "maplibregl-ctrl-group"
        );
        this.toggleButton = document.createElement("button");
        this.buttonContainer.appendChild(this.toggleButton);
        this.iconSpan = document.createElement("i");
        this.iconSpan.classList.add("fa", "fa-chart-line", "text-black");
        this.toggleButton.appendChild(this.iconSpan);
        this.toggleButton.addEventListener("click", this.toggleProfile.bind(this));
        const mapContainer = map.getContainer();
        const size = this.settings.size ?? "30%";

        if (this.settings.container) {
            const tmpContainer =
                typeof this.settings.container === "string"
                    ? document.getElementById(this.settings.container)
                    : this.settings.container;
            if (!tmpContainer) throw new Error("The provided container is invalid");
            this.profileContainer = tmpContainer as HTMLDivElement;
        } else {
            this.profileContainer = document.createElement("div");
            this.profileContainer.style.setProperty("display", "none");

            if (!this.settings.containerClass) {
                this.profileContainer.classList.add(this.settings.backgroundColor ?? "white")

                this.profileContainer.style.setProperty("position", "absolute");

                if (this.settings.position === "bottom" || !this.settings.position) {
                    // To prevent clashing with MapTiler logo and attribution control
                    this.settings.paddingBottom = this.settings.paddingBottom ?? 35;
                    this.profileContainer.style.setProperty("width", "100%");
                    this.profileContainer.style.setProperty("height", size);
                    this.profileContainer.style.setProperty("bottom", "0");
                } else if (this.settings.position === "top") {
                    this.profileContainer.style.setProperty("width", "100%");
                    this.profileContainer.style.setProperty("height", size);
                    this.profileContainer.style.setProperty("top", "0");
                } else if (this.settings.position === "left") {
                    // To prevent clashing with MapTiler logo and attribution control
                    this.settings.paddingBottom = this.settings.paddingBottom ?? 35;
                    this.profileContainer.style.setProperty("width", size);
                    this.profileContainer.style.setProperty("height", "100%");
                    this.profileContainer.style.setProperty("left", "0");
                } else if (this.settings.position === "right") {
                    // To prevent clashing with MapTiler logo and attribution control
                    this.settings.paddingBottom = this.settings.paddingBottom ?? 35;
                    this.profileContainer.style.setProperty("width", size);
                    this.profileContainer.style.setProperty("height", "100%");
                    this.profileContainer.style.setProperty("right", "0");
                }
            }

            mapContainer.appendChild(this.profileContainer);
        }

        const waypointContainer = document.createElement("div")
        waypointContainer.className = "absolute w-full"
        waypointContainer.id = "waypoint-container"

        this.profileContainer.append(waypointContainer);

        if (this.settings.containerClass) {
            this.profileContainer.classList.add(this.settings.containerClass);
        }

        this.elevationProfileChart = new ElevationProfile(
            this.profileContainer,
            this.settings
        );

        if (this.settings.visible) {
            this.showProfile();
        }

        return this.buttonContainer;
    }

    toggleProfile() {
        if (!this.profileContainer) return;

        if (this.isProfileShown) {
            this.hideProfile();
        } else {
            this.showProfile();
        }
        this.settings.onToggle?.(this.isProfileShown);
    }

    showProfile() {
        this.profileContainer?.style.setProperty("display", "inherit");
        this.isProfileShown = true;
    }

    hideProfile() {
        this.profileContainer?.style.setProperty("display", "none");
        this.isProfileShown = false;
    }

    moveCrosshair(lat: number, lon: number) {
        if (!this.elevationProfileChart) {
            return;
        }
        const chart = this.elevationProfileChart.chart
        const point = this.elevationProfileChart.getChartCoordinatesFromPosition(lat, lon)
        if (point == null) {
            return;
        }
        const rectangle = chart.canvas.getBoundingClientRect();

        const mouseMoveEvent = new MouseEvent('mousemove', {
            clientX: rectangle.left + point[0],
            clientY: rectangle.top + point[1]
        });

        chart.canvas.dispatchEvent(mouseMoveEvent);
    }

    hideCrosshair() {
        const mouseOutEvent = new MouseEvent('mouseout');
        return this.elevationProfileChart?.chart.canvas.dispatchEvent(mouseOutEvent);
    }

    onRemove(): void {
        // remove button
        if (this.buttonContainer?.parentNode) {
            this.buttonContainer.parentNode.removeChild(this.buttonContainer);
        }
        this.map = undefined;
        this.buttonContainer = undefined;
        this.toggleButton = undefined;
        this.isProfileShown = false;
    }

    getDefaultPosition?: (() => ControlPosition) | undefined;

    async setData(data: GeoJsonObject, waypoints?: Waypoint[]) {
        if (!this.map || !this.elevationProfileChart) {
            throw new Error(
                "The Elevation Profile Control needs to be mounted on a map instance before setting any data."
            );
        }

        this.data = data;

        if (!this.data) return;

        this.elevationProfileChart.setData(this.data, waypoints);
    }
}
