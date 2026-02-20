import type {
    Feature,
    FeatureCollection,
    GeoJsonObject,
    GeometryObject,
    LineString,
    MultiLineString,
    Position,
} from "geojson";

import { Chart, registerables } from "chart.js";
import zoomPlugin from "chartjs-plugin-zoom";
// @ts-ignore
import { CrosshairPlugin } from "chartjs-plugin-crosshair";

import { haversineDistance } from "$lib/models/gpx/utils";
import type { Waypoint } from "$lib/models/waypoint";
import { formatTimeHHMM } from "$lib/util/format_util";
import { haversineCumulatedDistanceWgs84, smoothElevations } from "./tools";

const FEET_PER_METER = 3.28084;
const MILES_PER_METER = 0.000621371;
const KILOMETERS_HOUR_PER_METER_SECOND = 3.6
const MILES_HOUR_PER_METER_SECOND = 2.23694

function extractLineStrings(
    geoJson: GeoJsonObject
): { lineStrings: Array<LineString | MultiLineString>, times: Date[] } {
    const lineStrings: Array<LineString | MultiLineString> = [];
    const times: Date[] = [];

    function extractFromGeometry(geometry: GeometryObject) {
        if (geometry.type === "LineString" || geometry.type === "MultiLineString") {
            lineStrings.push(geometry as LineString | MultiLineString);
        }
    }

    function extractFromFeature(feature: Feature) {
        if (feature.geometry) {
            extractFromGeometry(feature.geometry);
        }
        if (feature.properties?.coordinateProperties?.times) {
            const coordinateTimes = feature.properties?.coordinateProperties?.times.map((t: string) => new Date(t))
            times.push(...coordinateTimes)
        }
    }

    function extractFromFeatureCollection(collection: FeatureCollection) {
        for (const feature of collection.features) {
            if (feature.type === "Feature") {
                extractFromFeature(feature);
            } else if (feature.type === "FeatureCollection") {
                extractFromFeatureCollection(feature as unknown as FeatureCollection); // had to add unknown
            }
        }
    }

    if (geoJson.type === "Feature") {
        extractFromFeature(geoJson as Feature);
    } else if (geoJson.type === "FeatureCollection") {
        extractFromFeatureCollection(geoJson as FeatureCollection);
    } else {
        // It's a single geometry
        extractFromGeometry(geoJson as GeometryObject);
    }

    return { lineStrings, times };
}

function geoJsonObjectToPositionsAndTimes(geoJson: GeoJsonObject): { positions: Position[], times: Date[] } {
    const { lineStrings, times } = extractLineStrings(geoJson);
    const positionsGroups: Position[][] = [];

    for (let i = 0; i < lineStrings.length; i += 1) {
        const feature = lineStrings[i];
        if (feature.type === "LineString") {
            positionsGroups.push(feature.coordinates);
        } else if (feature.type === "MultiLineString") {
            positionsGroups.push(feature.coordinates.flat());
        }
    }
    return { positions: positionsGroups.flat(), times };
}

/**
 * Event data to `onMove` and `onClick` callback
 */
export type CallbackData = {
    /**
     * The position as `[lon, lat, elevation]`.
     * Elevation will be in meters if the component has been set with the unit "metric" (default)
     * of in feet if the unit is "imperial".
     */
    position: Position;
    /**
     * The distance from the start of the route. In km if the component has been set with the unit "metric" (default)
     * of in miles if the unit is "imperial".
     */
    distance: number;
    /**
     * Cumulated positive elevation from the begining of the route up to this location.
     * In meters if the component has been set with the unit "metric" (default)
     * of in feet if the unit is "imperial".
     */
    dPlus: number;
    /**
     * Slope grade in percentage (1% being a increase of 1m on a 100m distance)
     */
    gradePercent: number;
};

export type ElevationProfileOptions = {
    /**
     * Color of the background of the chart
     */
    backgroundColor?: string | null;
    /**
     * Unit system to use.
     * If "metric", elevation and D+ will be in meters, distances will be in km.
     * If "imperial", elevation and D+ will be in feet, distances will be in miles.
     *
     * Default: "metric"
     */
    unit?: "metric" | "imperial";
    /**
     * Font size applied to axes labels and tooltip.
     *
     * Default: `12`
     */
    fontSize?: number;
    /**
     * If `true`, will force the computation of the elevation of the GeoJSON data provided to the `.setData()` method,
     * even if they already contain elevation (possibly from GPS while recording). If `false`, the elevation will only
     * be computed if missing from the positions.
     *
     * Default: `false`
     */
    forceComputeElevation?: boolean;
    /**
     * Display the elevation label along the vertical axis.
     *
     * Default: `true`
     */
    displayElevationLabels?: boolean;
    /**
     * Display the distance labels alon the horizontal axis.
     *
     * Default: `true`
     */
    displayDistanceLabels?: boolean;
    /**
     * Display the distance and elevation units alongside the labels.
     *
     * Default: `true`
     */
    displayUnits?: boolean;
    /**
     * Color of the elevation and distance labels.
     *
     * Default: `"#0009"` (partially transparent black)
     */
    labelColor?: string;
    /**
     * Color of the elevation profile line.
     * Can be `null` to not display the line and rely on the background color only.
     *
     * Default: `"#66ccff"`
     */
    profileLineColor?: string | null;
    /**
     * Width of the elevation profile line.
     *
     * Default: `1.5`
     */
    profileLineWidth?: number;
    /**
     * Color of the elevation profile background (below the profile line)
     * Can be `null` to not display any backgound color.
     *
     * Default: `"#66ccff22"`
     */
    profileBackgroundColor?: string | null;
    /**
     * Display the tooltip folowing the pointer.
     *
     * Default: `true`
     */
    displayTooltip?: boolean;
    /**
     * Color of the text inside the tooltip.
     *
     * Default: `"#fff"`
     */
    tooltipTextColor?: string;
    /**
     * Color of the tooltip background.
     *
     * Default: `"#000A"` (partially transparent black)
     */
    tooltipBackgroundColor?: string;
    /**
     * Display the distance information inside the tooltip if `true`.
     *
     * Default: `true`
     */
    tooltipDisplayDistance?: boolean;
    /**
     * Display the elevation information inside the tooltip if `true`.
     *
     * Default: `true`
     */
    tooltipDisplayElevation?: boolean;
    /**
     * Display the D+ (cumulated positive ascent) inside the tooltip if `true`.
     *
     * Default: `true`
     */
    tooltipDisplayDPlus?: boolean;
    /**
     * Display the slope grade in percentage inside the tooltip if `true`.
     *
     * Default: `true`
     */
    tooltipDisplayGrade?: boolean;
    /**
    * Display the slope grade in percentage inside the tooltip if `true`.
    *
    * Default: `true`
    */
    tooltipDisplaySpeed?: boolean;
    /**
     * Display the distance grid lines (vertical lines matching the distance labels) if `true`.
     *
     * Default: `false`
     */
    displayDistanceGrid?: boolean;
    /**
     * Display the elevation grid lines (horizontal lines matching the elevation labels) if `true`.
     *
     * Default: `true`
     */
    displayElevationGrid?: boolean;
    /**
     * Color of the distance grid lines.
     *
     * Default: `"#0001"` (partially transparent black)
     */
    distanceGridColor?: string;
    /**
     * Color of the elevation drig lines.
     *
     * Default: `"#0001"` (partially transparent black)
     */
    elevationGridColor?: string;
    /**
     * Padding at the top of the chart, in number of pixels.
     *
     * Default: `30`
     */
    paddingTop?: number;
    /**
     * Padding at the bottom of the chart, in number of pixels.
     *
     * Default: `10`
     */
    paddingBottom?: number;
    /**
     * Padding at the left of the chart, in number of pixels.
     *
     * Default: `10`
     */
    paddingLeft?: number;
    /**
     * Padding at the right of the chart, in number of pixels.
     *
     * Default: `10`
     */
    paddingRight?: number;
    /**
     * Display the crosshair, a vertical line that follows the pointer, if `true`.
     *
     * Default: `true`
     */
    displayCrosshair?: boolean;
    /**
     * Color of the crosshair.
     *
     * Default: `"#0005"` (partially transparent black)
     */
    crosshairColor?: string;
    zoom?: boolean;
    /**
     * Callback function to call when the chart is zoomed or panned.
     * The argument `windowedLineString` is the GeoJSON LineString corresponding
     * to the portion of the route visible in the elevation chart.
     *
     * Default: `null`
     */
    onChangeView?: ((windowedLineString: LineString) => void) | null;
    /**
     * Callback function to call when the the elevation chart is clicked.
     *
     * Default: `null`
     */
    onClick?: ((data: CallbackData) => void) | null;
    /**
     * Callback function to call when the pointer is moving on the elevation chart.
     *
     * Default: `null`
     */
    onMove?: ((data: CallbackData) => void) | null;

    onEnter?: (() => void) | null;

    onLeave?: (() => void) | null;

};

const elevationProfileDefaultOptions: ElevationProfileOptions = {
    backgroundColor: null,
    unit: "metric",
    fontSize: 12,
    forceComputeElevation: false,
    displayElevationLabels: true,
    displayDistanceLabels: true,
    displayUnits: true,
    labelColor: "#0009",
    profileLineColor: "#66ccff",
    profileLineWidth: 1.5,
    profileBackgroundColor: "#66ccff22",
    displayTooltip: true,
    tooltipTextColor: "#fff",
    tooltipBackgroundColor: "#000A",
    tooltipDisplayDistance: true,
    tooltipDisplayElevation: true,
    tooltipDisplayDPlus: true,
    tooltipDisplayGrade: true,
    tooltipDisplaySpeed: true,
    displayDistanceGrid: false,
    displayElevationGrid: true,
    distanceGridColor: "#0001",
    elevationGridColor: "#0001",
    displayCrosshair: true,
    crosshairColor: "#0005",
    onChangeView: null,
    paddingTop: 30,
    paddingBottom: 10,
    paddingLeft: 10,
    paddingRight: 10,
    onClick: null,
    onMove: null,
    zoom: true
};

/**
 * Elevation profile chart
 */
export class ElevationProfile {
    private canvas: HTMLCanvasElement;
    private settings: ElevationProfileOptions;
    public chart: Chart<"line", Array<number>, number>;
    private elevatedPositions: Position[] = [];
    private elevatedPositionsAdjustedUnit: Position[] = [];
    private cumulatedDistance: number[] = [];
    private cumulatedDistanceAdjustedUnit: number[] = [];
    private cumulatedDPlus: number[] = [];
    private grade: number[] = [];
    private waypointPositions: number[] = []
    private waypoints: Waypoint[] = [];
    private times: Date[] = [];
    private cumulatedTime: number[] = []
    private speed: number[] = [];

    private gradeColor = [
        "#0d0887", // 0% and less
        "#3e049c", // 1
        "#6300a7", // 2
        "#8606a6", // 3
        "#a62098", // 4
        "#c03a83", // 5
        "#d5546e", // 6
        "#e76f5a", // 7
        "#f68d45", // 8
        "#fdae32", // 9
        "#fcd225", // 10
        "#f0f921", // more than 10%
    ]

    private width?: number
    private height?: number
    private gradient?: CanvasGradient;


    constructor(
        /**
         * DIV element to place the chart into
         */
        container: HTMLDivElement | string,
        /**
         * Options
         */
        options: ElevationProfileOptions = {}
    ) {
        const appContainer =
            typeof container === "string"
                ? document.getElementById(container)
                : container;
        if (!appContainer) {
            throw new Error("The container does not exist.");
        }

        this.canvas = document.createElement("canvas");
        this.canvas.id = "elevation-profile-chart"
        appContainer.appendChild(this.canvas);

        Chart.register(...registerables);
        Chart.register(zoomPlugin);

        this.settings = {
            ...elevationProfileDefaultOptions,
            ...options,
        };

        const distanceUnit = this.settings.unit === "imperial" ? "mi" : "km";
        const elevationUnit = this.settings.unit === "imperial" ? "ft" : "m";
        Chart.defaults.font.size = this.settings.fontSize;

        // using CrosshairPlugin normally can lead to race conditions:
        // https://github.com/AbelHeinsbroek/chartjs-plugin-crosshair/issues/119
        const CustomCrosshairPlugin = function (plugin: typeof CrosshairPlugin) {
            const originalAfterDraw = plugin.afterDraw;
            plugin.afterDraw = function (chart: Chart & { crosshair?: typeof CrosshairPlugin }, easing: boolean) {
                if (chart && chart.crosshair) {
                    originalAfterDraw.call(this, chart, easing);
                }
            };
            return plugin;
        };
        Chart.register(CustomCrosshairPlugin(CrosshairPlugin));

        this.chart = new Chart<"line", Array<number>, number>(this.canvas, {
            type: "line",

            data: {
                labels: [],
                datasets: [
                    {
                        label: "Elevation",
                        yAxisID: "y",
                        data: [],
                        pointRadius: 0,
                        fill: !!this.settings.profileBackgroundColor,
                        borderColor: (context) => {
                            const chart = context.chart;
                            return this.gradientFromElevation(chart)
                        },
                        // borderColor: this.settings.profileLineColor ?? "#0000",
                        backgroundColor: this.settings.profileBackgroundColor ?? "#0000",
                        tension: 0.1,
                        spanGaps: true,

                        // If line color is null, the line width is set to 0
                        borderWidth: this.settings.profileLineColor
                            ? this.settings.profileLineWidth
                            : 0,
                    }
                ],
            },

            options: {
                layout: {
                    padding: {
                        left: this.settings.paddingLeft,
                        right: this.settings.paddingRight,
                        bottom: this.settings.paddingBottom,
                        top: this.settings.paddingTop,
                    },
                },
                onClick: (_e, item) => {
                    if (typeof this.settings.onClick !== "function") return;

                    try {
                        const i = item[0].index;

                        this.settings.onClick.apply(this, [
                            {
                                position: this.elevatedPositionsAdjustedUnit[i],
                                distance: this.cumulatedDistanceAdjustedUnit[i],
                                dPlus: this.cumulatedDPlus[i],
                                gradePercent: this.grade[i],
                            },
                        ]);
                    } catch (e) {
                        // Nothing to do
                    }
                },
                onHover: (_e, item) => {
                    if (typeof this.settings.onMove !== "function") return;
                    try {
                        const i = item[0].index;

                        this.settings.onMove.apply(this, [
                            {
                                position: this.elevatedPositionsAdjustedUnit[i],
                                distance: this.cumulatedDistanceAdjustedUnit[i],
                                dPlus: this.cumulatedDPlus[i],
                                gradePercent: this.grade[i],
                            },
                        ]);
                    } catch (e) {
                        // Nothing to do
                    }
                },
                animation: false,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        min: 0,
                        max: 0,
                        type: "linear",
                        grid: {
                            display: this.settings.displayDistanceGrid,
                            drawOnChartArea: false,
                            color: this.settings.distanceGridColor,
                            drawTicks: true,
                            tickLength: 5,
                            tickColor: "#0002"
                        },
                        ticks: {
                            stepSize: 0.1,
                            align: "inner",
                            display: this.settings.displayDistanceLabels,
                            color: this.settings.labelColor,
                            maxRotation: 0,
                            callback: (value, index) => {
                                if (index % 10 !== 0) {
                                    return "";
                                }
                                const roundedValue = ~~((value as number) * 100) / 100;
                                return this.settings.displayUnits
                                    ? `${roundedValue} ${distanceUnit}`
                                    : roundedValue;
                            },
                        },
                    },
                    y: {
                        min: 0,
                        max: 0,
                        type: "linear",
                        afterTickToLabelConversion: (scaleInstance) => {
                            scaleInstance.ticks.pop();
                            scaleInstance.ticks.shift();
                        },
                        ticks: {
                            mirror: true,
                            maxTicksLimit: 10,
                            align: "end",
                            display: this.settings.displayElevationLabels,
                            color: this.settings.labelColor,

                            callback: (value) => {
                                const roundedValue = ~~((value as number) * 100) / 100;
                                return this.settings.displayUnits
                                    ? `${roundedValue} ${elevationUnit}`
                                    : roundedValue;
                            },
                        },
                        border: {
                            dash: [5, 5],
                            display: true,
                            color: this.settings.elevationGridColor,
                        },
                        grid: {
                            display: this.settings.displayElevationGrid,
                            color: this.settings.elevationGridColor,
                            drawTicks: false,
                        },
                    },
                },

                interaction: {
                    intersect: false,
                    mode: "index",
                },

                plugins: {
                    zoom: {
                        zoom: {
                            wheel: {
                                enabled: this.settings.zoom,
                            },
                            pinch: {
                                enabled: this.settings.zoom,
                            },
                            mode: "x",
                        },
                        pan: {
                            enabled: this.settings.zoom,
                            mode: "x",
                        },
                        limits: {
                            x: {
                                min: "original",
                                max: "original",
                            },
                        },
                    },
                    title: {
                        display: false,
                    },
                    legend: {
                        display: false,
                    },
                    tooltip: {
                        enabled: this.settings.displayTooltip,
                        yAlign: "center",
                        cornerRadius: 3,
                        displayColors: false,
                        backgroundColor: this.settings.tooltipBackgroundColor,
                        bodyColor: this.settings.tooltipTextColor,
                        callbacks: {
                            title: () => {
                                return "";
                            },

                            label: (tooltipItem) => {
                                if (tooltipItem.datasetIndex != 0) {
                                    return "";
                                }
                                const tooltipInfo = [];
                                if (this.settings.tooltipDisplayDistance) {
                                    tooltipInfo.push(
                                        `After: ${this.cumulatedDistanceAdjustedUnit[
                                            tooltipItem.dataIndex
                                        ].toFixed(2)} ${distanceUnit} ${this.cumulatedTime.length ? '(' + formatTimeHHMM(this.cumulatedTime[tooltipItem.dataIndex]) + ')' : ''}`
                                    );
                                }

                                if (this.settings.tooltipDisplayElevation) {
                                    tooltipInfo.push(
                                        `Elevation: ${this.elevatedPositionsAdjustedUnit[
                                            tooltipItem.dataIndex
                                        ][2].toFixed(2)} ${elevationUnit}`
                                    );
                                }

                                if (this.settings.tooltipDisplayDPlus) {
                                    tooltipInfo.push(
                                        `D+: ${this.cumulatedDPlus[tooltipItem.dataIndex].toFixed(
                                            0
                                        )} ${elevationUnit}`
                                    );
                                }

                                if (this.settings.tooltipDisplayGrade) {
                                    tooltipInfo.push(
                                        `Grade: ${this.grade[tooltipItem.dataIndex].toFixed(1)}%`
                                    );
                                }

                                if (this.settings.tooltipDisplaySpeed && this.speed.length) {
                                    tooltipInfo.push(`Speed: ${this.speed[
                                        tooltipItem.dataIndex
                                    ]?.toFixed(2)} ${distanceUnit}/h`
                                    );
                                }

                                return tooltipInfo;
                            },
                        },
                    },

                    // The crosshair plugin does not have types
                    // @ts-ignore
                    crosshair: {
                        zoom: {
                            enabled: false,
                        },
                        line: {
                            color: this.settings.displayCrosshair
                                ? this.settings.crosshairColor
                                : "#0000",
                            width: 1,
                        },
                    },
                },
            },

            plugins: [
                {
                    id: "waypointPlugin",
                    afterDraw: (chart, args, options) => {
                        const waypointContainer = document.getElementById("waypoint-container") as HTMLDivElement;
                        waypointContainer.innerHTML = ""; // Clear previous ticks

                        const xScale = chart.scales.x; // Get X-axis scale
                        const chartRect = chart.canvas.getBoundingClientRect(); // Canvas position

                        this.waypointPositions.forEach((position: number, index: number) => {

                            const xPos = xScale.getPixelForValue(position); // X-axis pixel for tick

                            // Create custom HTML tick
                            const wpDiv = document.createElement("div");
                            wpDiv.className = "wp-marker absolute -translate-x-1/2 w-6 aspect-square bg-background-inverse rounded-full flex justify-center items-center text-content-inverse cursor-pointer hover:scale-110";
                            wpDiv.style.left = `${xPos}px`; // Position horizontally
                            wpDiv.style.top = `8px`; // Position horizontally

                            // Add custom HTML content (e.g., icon + label)
                            wpDiv.innerHTML = `<div class="tooltip" data-title="${this.waypoints[index]?.name ?? "?"}"><i class="fa fa-${this.waypoints.at(index)?.icon ?? 'circle'}"></i></div>`;

                            waypointContainer.appendChild(wpDiv); // Add to container
                        });
                    }
                },
                {
                    id: "customZoomEvent",
                    afterDataLimits: (chart) => {
                        if (typeof this.settings.onChangeView !== "function") return;
                        try {
                            this.settings.onChangeView.apply(this, [
                                this.createWindowExtractLineString(),
                            ]);
                        } catch (e) {
                            // nothing to do
                        }
                    },
                },
            ],
        });
        if (typeof this.settings.onLeave === "function") {
            this.chart.canvas.addEventListener("mouseout", this.settings.onLeave)
        }

        if (typeof this.settings.onEnter === "function") {
            this.chart.canvas.addEventListener("mouseenter", this.settings.onEnter)
        }

        // If the tooltip is shown, then we hide it when panning the chart
        if (this.settings.displayTooltip) {
            let mouseDown = false;
            this.chart.canvas.addEventListener("mousedown", () => {
                mouseDown = true;
            });

            this.chart.canvas.addEventListener("mousemove", () => {
                if (
                    mouseDown &&
                    this.chart.options.plugins &&
                    this.chart.options.plugins.tooltip
                ) {
                    this.chart.options.plugins.tooltip.enabled = false;
                    this.chart.update();
                }
            });

            window.addEventListener("mouseup", () => {
                if (this.chart.options.plugins && this.chart.options.plugins.tooltip) {
                    this.chart.options.plugins.tooltip.enabled = true;
                    this.chart.update();
                    mouseDown = false;
                }
            });
        }
    }

    getChartCoordinatesFromPosition(lat: number, lon: number) {

        let minDistance = Infinity
        let bestCandiateIndex: number = -1
        for (let i = 0; i < this.elevatedPositionsAdjustedUnit.length; i++) {
            const p = this.elevatedPositionsAdjustedUnit[i];
            const deltaLat = lat - p[1];
            const deltaLon = lon - p[0];
            const distance = deltaLat * deltaLat + deltaLon * deltaLon;

            if (distance < minDistance) {
                minDistance = distance;
                bestCandiateIndex = i
            }
        };

        if (bestCandiateIndex < 0) {
            return null;
        }
        const meta = this.chart.getDatasetMeta(0);
        const element = meta.data[bestCandiateIndex];
        if (element) {
            return [element.x, element.y];
        }
        return null;
    }

    gradientFromElevation(chart: Chart, force: boolean = false) {
        const ctx = chart.ctx;
        const chartArea = chart.chartArea;

        if (!chartArea) {
            return;
        }

        const chartWidth = chartArea.right - chartArea.left;
        const chartHeight = chartArea.bottom - chartArea.top;

        if (chartWidth == 0 || chartHeight == 0) {
            return;
        }

        if (!this.gradient || this.width !== chartWidth || this.height !== chartHeight || force) {
            this.width = chartWidth;
            this.height = chartHeight;

            this.gradient = ctx.createLinearGradient(chartArea.left, 0, chartArea.right, 0) as CanvasGradient;

            this.gradient.addColorStop(0, this.gradeColor[0])

            let prevColor = this.gradeColor[0];
            for (let i = 0; i < this.grade.length; i++) {
                const grade = ~~(Math.abs(this.grade[i]) / 2.5);

                let color;
                if (grade < 1) {
                    color = this.gradeColor[0]
                }
                else if (grade > 10) {
                    color = this.gradeColor[11];
                } else {
                    color = this.gradeColor[grade];
                }


                if (color !== prevColor) {
                    const percentDone = this.cumulatedDistance[i] / this.cumulatedDistance[this.cumulatedDistance.length - 1]
                    this.gradient.addColorStop(percentDone, color);

                    prevColor = color;
                }

            }
        }
        return this.gradient;
    }

    createWindowExtractLineString(): LineString {
        const scaleMin = this.chart.scales.x.min;
        const scaleMax = this.chart.scales.x.max;

        const cda = this.cumulatedDistanceAdjustedUnit;
        const nbElem = cda.length;

        let indexStart = 0;
        let indexEnd = nbElem - 1;

        // find the start index
        for (let i = 0; i < nbElem; i += 1) {
            if (cda[i] >= scaleMin) {
                indexStart = i;
                break;
            }
        }

        // find the end index
        for (let i = nbElem - 1; i >= indexStart; i -= 1) {
            if (cda[i] <= scaleMax) {
                indexEnd = i;
                break;
            }
        }

        return this.createExtractLineString(indexStart, indexEnd);
    }

    createExtractLineString(fromIndex: number, toIndex: number): LineString {
        const elevatedPositionsWindow: Position[] = this.elevatedPositions.slice(
            fromIndex,
            toIndex
        );

        return {
            type: "LineString",
            coordinates: elevatedPositionsWindow,
        };
    }

    toggleTheme(options: ElevationProfileOptions) {
        this.settings = {
            ...this.settings,
            ...options,
        };
        this.chart.data.datasets[0].backgroundColor = this.settings.profileBackgroundColor ?? "#0000";
        this.chart.options.scales!.x!.ticks!.color = this.settings.labelColor;

        this.chart.options.scales!.y!.grid!.color = this.settings.elevationGridColor;
        this.chart.options.scales!.y!.border!.color = this.settings.elevationGridColor;
        this.chart.options.scales!.y!.ticks!.color = this.settings.labelColor;

        (this.chart.options.plugins as any).crosshair!.line.color = this.settings.crosshairColor;

        this.chart.update();
    }

    async setData(data: GeoJsonObject, waypoints?: Waypoint[]) {
        // Concatenates the positions that may come from multiple LineStrings or MultiLineString
        const { positions, times } = geoJsonObjectToPositionsAndTimes(data);

        this.times = times;

        this.elevatedPositions = smoothElevations(positions, Math.ceil(positions.length / 100));

        this.cumulatedDistance = haversineCumulatedDistanceWgs84(
            this.elevatedPositions
        );

        // Conversion of distance to miles and elevation to feet
        if (this.settings.unit === "imperial") {
            this.cumulatedDistanceAdjustedUnit = this.cumulatedDistance.map(
                (dist) => dist * MILES_PER_METER
            );
            this.elevatedPositionsAdjustedUnit = this.elevatedPositions.map((pos) => [
                pos[0],
                pos[1],
                pos[2] * FEET_PER_METER,
            ]);
            this.cumulatedDPlus = this.cumulatedDPlus.map(
                (ele) => ele * FEET_PER_METER
            );
        } else {
            this.cumulatedDistanceAdjustedUnit = this.cumulatedDistance.map(
                (dist) => dist / 1000
            ); // we still need to convert distance to km
            this.elevatedPositionsAdjustedUnit = this.elevatedPositions;
        }

        this.cumulatedDPlus = [];
        this.grade = [];
        this.waypoints = waypoints ?? [];
        this.waypointPositions = [];

        let cumulatedDPlus = 0;
        let cumulatedTime = 0;

        const minSegmentDistance = (this.cumulatedDistance.at(-1) ?? 1000) / 100;
        let segmentStartIndex = 0;

        // Initialize an array to store the minimum distance for each waypoint
        const minDistances = new Array(waypoints?.length).fill(Infinity);

        for (let i = 0; i < this.elevatedPositions.length; i++) {

            // Check if a waypoint is closest to this point
            this.waypoints.forEach((waypoint, waypointIndex) => {
                const distance = haversineDistance(this.elevatedPositions[i][1], this.elevatedPositions[i][0], waypoint.lat, waypoint.lon);

                // Update if the current route coordinate is closer to the waypoint
                if (distance < minDistances[waypointIndex]) {
                    minDistances[waypointIndex] = distance;
                    this.waypointPositions[waypointIndex] = this.cumulatedDistanceAdjustedUnit[i];
                    waypoint.distance_from_start = this.cumulatedDistanceAdjustedUnit[i] * 1000;
                }
            });

            const elevation = this.elevatedPositions[i][2];
            const time = this.times[i]
            if (i > 1) {
                const elevationPrevious = this.elevatedPositions[i - 1][2];
                const elevationDelta = elevation - elevationPrevious;
                const segmentDistance =
                    this.cumulatedDistance[i] - this.cumulatedDistance[segmentStartIndex];
                cumulatedDPlus += Math.max(0, elevationDelta);
                this.cumulatedDPlus.push(cumulatedDPlus);

                if (time) {
                    const timePrevious = this.times[i - 1];
                    const timeDelta = (time.getTime() - timePrevious.getTime()) / (1000);
                    cumulatedTime += timeDelta;
                    this.cumulatedTime.push(cumulatedTime)
                }


                // Check if the segment distance is greater than or equal to the minimum threshold
                if (segmentDistance >= minSegmentDistance || i === this.elevatedPositions.length - 1) {
                    // Calculate the grade for the consolidated segment
                    const elevationStart = this.elevatedPositions[segmentStartIndex][2];
                    const elevationEnd = this.elevatedPositions[i][2];
                    const elevationDelta = elevationEnd - elevationStart;
                    const gradePercent = (elevationDelta / segmentDistance) * 100; // Grade as a percentage

                    let speed;
                    if (this.times.length) {
                        const distanceStart = this.cumulatedDistance[segmentStartIndex]
                        const distanceEnd = this.cumulatedDistance[i]
                        const distanceDelta = distanceEnd - distanceStart

                        const timeStart = this.times[segmentStartIndex]
                        const timeEnd = this.times[i]
                        if (timeStart && timeEnd) {
                            const timeDelta = (timeEnd.getTime() - timeStart.getTime()) / 1000

                            speed = (distanceDelta / timeDelta)
                            if (this.settings.unit === "imperial") {
                                speed = speed * MILES_HOUR_PER_METER_SECOND
                            } else {
                                speed = speed * KILOMETERS_HOUR_PER_METER_SECOND;
                            }
                        }

                    }


                    // Apply the same grade to all positions within this segment
                    for (let j = segmentStartIndex; j <= i; j++) {
                        this.grade.push(gradePercent);
                        if (speed) {
                            this.speed.push(speed)
                        }
                    }

                    // Move to the next segment
                    segmentStartIndex = i + 1;
                }
            }
        }

        this.grade.push(this.grade.at(-1) ?? 0);
        this.cumulatedDPlus.push(cumulatedDPlus);


        let minElevation = +Infinity;
        let maxElevation = -Infinity;

        for (let i = 0; i < this.elevatedPositionsAdjustedUnit.length; i += 1) {
            if (this.elevatedPositionsAdjustedUnit[i][2] < minElevation) {
                minElevation = this.elevatedPositionsAdjustedUnit[i][2];
            }

            if (this.elevatedPositionsAdjustedUnit[i][2] > maxElevation) {
                maxElevation = this.elevatedPositionsAdjustedUnit[i][2];
            }
        }

        const elevationPadding = (maxElevation - minElevation) * 0.1;
        this.chart.data.labels = this.cumulatedDistanceAdjustedUnit;
        this.chart.data.datasets[0].data = this.elevatedPositionsAdjustedUnit.map(
            (pos) => pos[2]
        );

        const gradient = this.gradientFromElevation(this.chart, true);
        if (gradient) {
            this.chart.data.datasets[0].borderColor = gradient
        }

        if (
            this.chart.options.scales &&
            this.chart.options.scales.x &&
            this.chart.options.scales.y
        ) {
            this.chart.options.scales.x.min = this.cumulatedDistanceAdjustedUnit[0];
            this.chart.options.scales.x.max =
                this.cumulatedDistanceAdjustedUnit[
                this.cumulatedDistanceAdjustedUnit.length - 1
                ];


            this.chart.options.scales.y.min = minElevation - elevationPadding;
            this.chart.options.scales.y.max = maxElevation + elevationPadding;
        }
        this.chart.update();
    }

}