<script lang="ts">
    import { page } from "$app/state";
    import Calendar from "$lib/components/base/calendar.svelte";
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import MultiSelect from "$lib/components/base/multi_select.svelte";
    import Select, {
        type SelectItem,
    } from "$lib/components/base/select.svelte";
    import SummitLogTable from "$lib/components/summit_log/summit_log_table.svelte";
    import type { SummitLog } from "$lib/models/summit_log.js";
    import { categories } from "$lib/stores/category_store.js";
    import { profile_stats_index } from "$lib/stores/profile_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        formatDistance,
        formatElevation,
        formatSpeed,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import Bar from "$lib/vendor/svelte-chartjs/bar.svelte";
    import Pie from "$lib/vendor/svelte-chartjs/pie.svelte";
    import {
        ArcElement,
        BarElement,
        CategoryScale,
        Chart as ChartJS,
        Legend,
        LinearScale,
        Title,
        Tooltip,
    } from "chart.js";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    ChartJS.register(
        Title,
        Tooltip,
        Legend,
        ArcElement,
        CategoryScale,
        LinearScale,
        BarElement,
    );

    let summitLogs: SummitLog[] = $state(data.logs);

    const filter = $state(data.filter);

    const categorySelectItems: SelectItem[] = $categories.map((c) => ({
        value: c.id,
        text: c.name,
    }));

    const barChartSelectItems: SelectItem[] = [
        {
            text: $_("distance"),
            value: "distance",
        },
        {
            text: $_("duration"),
            value: "duration",
        },
        {
            text: $_("elevation-gain"),
            value: "elevation_gain",
        },
        {
            text: $_("elevation-loss"),
            value: "elevation_loss",
        },
    ];

    let barChartSelectedValue = $state(barChartSelectItems[0].value);

    const categoryColors = [
        "#fb8500",
        "#ffb703",
        "#06d6a0",
        "#219ebc",
        "#8ecae6",
        "#ffafcc",
    ];

    const conversionFactors = {
        distance: 0.621371,
        elevation_gain: 3.28084,
        elevation_loss: 3.28084,
    };

    let logCategories = $derived(
        summitLogs.reduce(
            (acc, log) => {
                const cat =
                    log.expand?.trail?.expand?.category?.name ?? "unknown";
                acc[$_(cat)] = (acc[$_(cat)] || 0) + 1;
                return acc;
            },
            {} as Record<string, number>,
        ),
    );

    let categoryLabels = $derived(Object.keys(logCategories).sort());
    let categoryValues = $derived(Object.values(logCategories));
    let categoryColorMap = $derived(
        Object.fromEntries(
            categoryLabels.map((label, index) => [
                label,
                categoryColors[index % categoryColors.length],
            ]),
        ),
    );

    let categoryChartData = $derived({
        labels: categoryLabels,
        datasets: [
            {
                data: categoryValues,
                backgroundColor: categoryColors,
            },
        ],
    });

    function barChartUnit() {
        const unit = page.data.settings?.unit ?? "metric";

        switch (barChartSelectedValue) {
            case "duration":
                return "h";
            case "elevation_gain":
            case "elevation_loss":
                return unit == "metric" ? "m" : "ft";
            case "distance":
                return unit == "metric" ? "km" : "mi";
        }
    }

    let barChartDataByDate = $derived(
        summitLogs.reduce(
            (acc, log) => {
                const date = new Date(log.date).toLocaleDateString(undefined, {
                    month: "2-digit",
                    day: "2-digit",
                    year: "numeric",
                    timeZone: "UTC",
                });
                acc[date] =
                    (acc[date] || 0) +
                    ((log as any)[barChartSelectedValue] ?? 0);
                if (barChartSelectedValue === "distance") {
                    acc[date] = acc[date] / 1000;
                } else if (barChartSelectedValue === "duration") {
                    acc[date] = acc[date] / 60 / 60;
                }
                if (page.data.settings?.unit !== "metric") {
                    acc[date] =
                        acc[date] *
                        (conversionFactors as any)[barChartSelectedValue];
                }
                return acc;
            },
            {} as Record<string, number>,
        ),
    );

    let barChartLabels = $derived(Object.keys(barChartDataByDate)); // Dates as labels
    let barChartValues = $derived(Object.values(barChartDataByDate));

    let barChartData = $derived({
        labels: barChartLabels,
        datasets: [
            {
                label: barChartSelectItems.find(
                    (i) => i.value === barChartSelectedValue,
                )?.text,
                data: barChartValues,
                backgroundColor: "#3388ff",
                borderRadius: 15,
            },
        ],
    });

    let totalDistance = $derived(
        summitLogs.reduce((sum, log) => sum + (log.distance ?? 0), 0),
    );

    let totalDuration = $derived(
        summitLogs.reduce((sum, log) => sum + (log.duration ?? 0), 0),
    );

    let totalElevationGain = $derived(
        summitLogs.reduce((sum, log) => sum + (log.elevation_gain ?? 0), 0),
    );

    let totalElevationLoss = $derived(
        summitLogs.reduce((sum, log) => sum + (log.elevation_loss ?? 0), 0),
    );

    let averageSpeed = $derived(
        totalDuration > 0
            ? summitLogs.reduce(
                  (sum, log) =>
                      sum + (log.distance && log.duration ? log.distance : 0),
                  0,
              ) / totalDuration
            : undefined,
    );

    function updateFilterCategory(categories: SelectItem[]) {
        filter.category = categories.map((c) => c.value);
        loadSummitLogs();
    }

    function handleDateClick(date: Date) {
        const datePlusN = date;
        datePlusN.setDate(date.getDate() + 1);
        filter.startDate = datePlusN.toISOString().slice(0, 10);
        datePlusN.setDate(date.getDate() + 1);
        filter.endDate = datePlusN.toISOString().slice(0, 10);
        loadSummitLogs();
    }

    async function loadSummitLogs() {
        try {
            const logs = await profile_stats_index(page.params.handle!, filter);
            summitLogs = logs;
        } catch (e) {
            show_toast({
                icon: "close",
                text: "Error loading stats.",
                type: "error",
            });
        }
    }
</script>

<svelte:head>
    <title>{$_("profile")} | wanderer</title>
</svelte:head>

<div
    class="grid grid-cols-1 lg:grid-cols-[320px_minmax(0,_1fr)] gap-y-4 max-w-6xl mx-auto"
>
    <div
        class="flex flex-wrap lg:flex-nowrap col-start-1 lg:col-start-2 gap-x-4 justify-end"
    >
        <MultiSelect
            onchange={updateFilterCategory}
            label={$_("categories")}
            items={categorySelectItems}
            placeholder={`${$_("filter-categories")}...`}
        ></MultiSelect>
        <Datepicker
            onchange={loadSummitLogs}
            bind:value={filter.startDate}
            label={$_("after")}
        ></Datepicker>
        <Datepicker
            onchange={loadSummitLogs}
            bind:value={filter.endDate}
            label={$_("before")}
        ></Datepicker>
    </div>
    <div class="space-y-4 grow-0 lg:mr-4">
        <div class="border border-input-border rounded-xl p-6">
            <Calendar
                onclick={handleDateClick}
                logs={summitLogs}
                colorMap={categoryColorMap}
            ></Calendar>
        </div>
        <div class="border border-input-border rounded-xl p-6 space-y-4">
            <span class="text-gray-500 font-semibold text-lg"
                ><i class="fa fa-person-hiking mr-3"></i>{$_(
                    "categories",
                )}</span
            >
            <Pie
                data={categoryChartData}
                options={{
                    responsive: true,
                    plugins: {
                        legend: {
                            position: "bottom",
                        },
                    },
                }}
            />
        </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-hashtag mr-3"></i>{$_("activity", {
                    values: { n: 2 },
                })}</span
            >
            <p class="text-3xl font-bold">{summitLogs.length}</p>
        </div>
        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-left-right mr-3"></i>{$_("distance")}</span
            >
            <p class="text-3xl font-bold">{formatDistance(totalDistance)}</p>
        </div>

        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-clock mr-3"></i>{$_("duration")}</span
            >
            <p class="text-3xl font-bold">
                {formatTimeHHMM(totalDuration)}
            </p>
        </div>
        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-arrow-trend-down mr-3"></i>{$_(
                    "average-speed",
                )}</span
            >
            <p class="text-3xl font-bold">{formatSpeed(averageSpeed)}</p>
        </div>
        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-arrow-trend-up mr-3"></i>{$_(
                    "elevation-gain",
                )}</span
            >
            <p class="text-3xl font-bold">
                {formatElevation(totalElevationGain)}
            </p>
        </div>
        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-arrow-trend-down mr-3"></i>{$_(
                    "elevation-loss",
                )}</span
            >
            <p class="text-3xl font-bold">
                {formatElevation(totalElevationLoss)}
            </p>
        </div>

        <div
            class="h-full lg:col-span-2 space-y-2 border border-input-border rounded-xl p-6"
        >
            <div class="flex justify-between">
                <span class="text-gray-500 font-semibold text-lg"
                    ><i class="fa fa-calendar mr-3"></i>{$_("activity", {
                        values: { n: 1 },
                    })}</span
                >
                <Select
                    bind:value={barChartSelectedValue}
                    items={barChartSelectItems}
                ></Select>
            </div>

            <Bar
                data={barChartData}
                options={{
                    plugins: {
                        legend: {
                            display: false,
                        },
                        tooltip: {
                            callbacks: {
                                label: (item) =>
                                    `${item.dataset.label}: ${item.formattedValue} ${barChartUnit()}`,
                            },
                        },
                    },
                    scales: {
                        y: {
                            ticks: {
                                callback: function (value, index, ticks) {
                                    return value + " " + barChartUnit();
                                },
                            },
                        },
                    },
                }}
            ></Bar>
        </div>
    </div>

    <div
        class="col-span-1 lg:col-span-2 border border-input-border rounded-xl p-6 space-y-6"
    >
        <span class="text-gray-500 font-semibold text-lg"
            ><i class="fa fa-table mr-3"></i>{$_("all-activities")}</span
        >
        <div class=" overflow-x-auto">
            <SummitLogTable
                {summitLogs}
                handle={page.params.handle!}
                showCategory
                showTrail
                showRoute
            ></SummitLogTable>
        </div>
    </div>
</div>
