<script lang="ts">
    import { POINT_METRIC_KEYS, type PointMetricKey } from "$lib/models/gpx/extensions";
    import { haversineCumulatedDistanceWgs84 } from "$lib/vendor/maplibre-elevation-profile/tools";
    import type { Feature, FeatureCollection, Position } from "geojson";
    import { _ } from "svelte-i18n";
    import MetricChart from "./metric_chart.svelte";

    interface Props {
        geojson?: FeatureCollection;
        unit?: "metric" | "imperial";
    }

    let { geojson, unit = "metric" }: Props = $props();

    const MILES_PER_METER = 0.000621371;

    const metricMeta: Record<
        PointMetricKey,
        { labelKey: string; unit: (u: "metric" | "imperial") => string; color: string }
    > = {
        heartRate: { labelKey: "heart-rate", unit: () => "bpm", color: "#d62728" },
        power: { labelKey: "power", unit: () => "W", color: "#9467bd" },
        cadence: { labelKey: "cadence", unit: () => "rpm", color: "#2ca02c" },
        temperature: {
            labelKey: "temperature",
            unit: (u) => (u === "imperial" ? "°F" : "°C"),
            color: "#17becf",
        },
    };

    type MetricChartData = {
        key: PointMetricKey;
        label: string;
        unit: string;
        color: string;
        distances: number[];
        values: (number | null)[];
    };

    let charts: MetricChartData[] = $derived(buildCharts(geojson, unit));
    let distanceUnit = $derived(unit === "imperial" ? "mi" : "km");

    function buildCharts(
        data: FeatureCollection | undefined,
        u: "metric" | "imperial",
    ): MetricChartData[] {
        if (!data || data.type !== "FeatureCollection") {
            return [];
        }

        const positions: Position[] = [];
        const series: Record<PointMetricKey, (number | null)[]> = {
            heartRate: [],
            cadence: [],
            power: [],
            temperature: [],
        };
        const present: Record<PointMetricKey, boolean> = {
            heartRate: false,
            cadence: false,
            power: false,
            temperature: false,
        };

        for (const feature of data.features as Feature[]) {
            const geometry = feature.geometry;
            if (!geometry) {
                continue;
            }
            let coords: Position[] = [];
            if (geometry.type === "LineString") {
                coords = geometry.coordinates;
            } else if (geometry.type === "MultiLineString") {
                coords = geometry.coordinates.flat();
            } else {
                continue;
            }

            const cp = (feature.properties?.coordinateProperties ?? {}) as Record<string, unknown>;
            for (let i = 0; i < coords.length; i++) {
                positions.push(coords[i]);
                for (const key of POINT_METRIC_KEYS) {
                    const arr = cp[key];
                    const value = Array.isArray(arr) ? arr[i] : null;
                    const num = typeof value === "number" && Number.isFinite(value) ? value : null;
                    if (num !== null) {
                        present[key] = true;
                    }
                    series[key].push(num);
                }
            }
        }

        const presentKeys = POINT_METRIC_KEYS.filter((key) => present[key]);
        if (presentKeys.length === 0) {
            return [];
        }

        const cumulatedMeters = haversineCumulatedDistanceWgs84(positions);
        const distances = cumulatedMeters.map((m) =>
            u === "imperial" ? m * MILES_PER_METER : m / 1000,
        );

        return presentKeys.map((key) => {
            const meta = metricMeta[key];
            let values = series[key];
            if (key === "temperature" && u === "imperial") {
                values = values.map((v) => (v === null ? null : v * 1.8 + 32));
            }
            return {
                key,
                label: $_(meta.labelKey),
                unit: meta.unit(u),
                color: meta.color,
                distances,
                values,
            };
        });
    }
</script>

{#if charts.length}
    <div class="flex flex-col gap-3 mb-6">
        {#each charts as chart (chart.key)}
            <div>
                <p class="text-xs font-medium mb-1" style="color: {chart.color}">
                    {chart.label} ({chart.unit})
                </p>
                <div class="h-20">
                    <MetricChart
                        label={chart.label}
                        unit={chart.unit}
                        color={chart.color}
                        distances={chart.distances}
                        values={chart.values}
                        {distanceUnit}
                    ></MetricChart>
                </div>
            </div>
        {/each}
    </div>
{/if}
