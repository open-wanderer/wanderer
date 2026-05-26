<script lang="ts">
    import {
        Chart,
        Filler,
        LineController,
        LineElement,
        LinearScale,
        PointElement,
        Tooltip,
    } from "chart.js";

    interface Props {
        label: string;
        unit: string;
        color: string;
        distances: number[];
        values: (number | null)[];
        distanceUnit: string;
    }

    let { label, unit, color, distances, values, distanceUnit }: Props = $props();

    Chart.register(LineController, LineElement, PointElement, LinearScale, Filler, Tooltip);

    let canvas: HTMLCanvasElement;

    $effect(() => {
        const minX = distances[0] ?? 0;
        const maxX = distances[distances.length - 1] ?? 0;

        const chart = new Chart(canvas, {
            type: "line",
            data: {
                labels: distances,
                datasets: [
                    {
                        data: values as number[],
                        borderColor: color,
                        backgroundColor: color + "22",
                        borderWidth: 1.5,
                        pointRadius: 0,
                        tension: 0.1,
                        fill: true,
                        spanGaps: true,
                    },
                ],
            },
            options: {
                animation: false,
                maintainAspectRatio: false,
                interaction: { intersect: false, mode: "index" },
                scales: {
                    x: {
                        type: "linear",
                        min: minX,
                        max: maxX,
                        ticks: { display: false },
                        grid: { display: false },
                    },
                    y: {
                        type: "linear",
                        ticks: { maxTicksLimit: 4, font: { size: 10 } },
                        grid: { color: "#8884" },
                    },
                },
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        displayColors: false,
                        callbacks: {
                            title: (items) =>
                                items.length
                                    ? `${Number(items[0].parsed.x).toFixed(2)} ${distanceUnit}`
                                    : "",
                            label: (item) =>
                                `${label}: ${Math.round(Number(item.parsed.y))} ${unit}`,
                        },
                    },
                },
            },
        });

        return () => chart.destroy();
    });
</script>

<canvas bind:this={canvas} aria-label={`${label} (${unit})`}></canvas>
