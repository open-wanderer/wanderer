<script lang="ts">
    import { page } from "$app/state";
    import Calendar from "$lib/components/base/calendar.svelte";
    import DateRangeModal from "$lib/components/base/date_range_modal.svelte";
    import MultiSelect from "$lib/components/base/multi_select.svelte";
    import Select, {
        type SelectItem,
    } from "$lib/components/base/select.svelte";
    import SummitLogTable from "$lib/components/summit_log/summit_log_table.svelte";
    import TrailCategoryFilter from "$lib/components/trail/trail_category_filter.svelte";
    import type { StatisticActivity } from "$lib/models/statistic_activity.js";
    import type { SummitLogFilter } from "$lib/models/summit_log.js";
    import { categories } from "$lib/stores/category_store.js";
    import { profile_stats_index } from "$lib/stores/profile_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        displayCategoryIcon,
        displayCategoryName,
        displaySubcategoryLabel,
        displayTrailCategoryBadgeIcon,
        displayTrailCategoryIcon,
        displayTrailCategoryLabel,
        trailCategoryKey,
    } from "$lib/util/category_util";
    import {
        formatDistance,
        formatElevation,
        formatSpeed,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import {
        dateInputValue,
        datePeriodRange,
        datePeriodPresetForRange,
        monthDateRange,
        parseDateValue,
        type DatePeriodPreset,
    } from "$lib/util/date_util";
    import {
        activityChartBucketKey,
        activityChartBuckets,
        activityChartDataUnit,
        activityChartTickUnit,
        dateAxisValue,
        millisecondsPerDay,
        type ActivityChartBucket,
        type ActivityChartTimeUnit,
    } from "$lib/util/activity_chart_util";
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
        type ChartDataset,
    } from "chart.js";
    import { onMount, untrack } from "svelte";
    import { _, locale } from "svelte-i18n";

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

    const initialFilter: SummitLogFilter = untrack(() => data.filter);
    type PeriodMode = DatePeriodPreset | "custom";
    const initialPeriodMode: PeriodMode =
        datePeriodPresetForRange(
            initialFilter.startDate,
            initialFilter.endDate,
        ) ?? "custom";
    const initialCalendarRange = monthDateRange(
        initialFilter.startDate
            ? parseDateValue(initialFilter.startDate)
            : new Date(),
    );

    let activities: StatisticActivity[] = $state(untrack(() => data.activities));
    let calendarActivities: StatisticActivity[] = $state(
        untrack(() => data.activities),
    );

    const filter: SummitLogFilter = $state(initialFilter);
    let periodMode = $state<PeriodMode>(initialPeriodMode);
    let appliedPeriodMode = $state<PeriodMode>(initialPeriodMode);
    let calendarRange = $state(initialCalendarRange);
    let periodRangeModal: DateRangeModal;

    const periodSelectItems: SelectItem[] = [
        { text: $_("current-month"), value: "current_month" },
        { text: $_("current-quarter"), value: "current_quarter" },
        { text: $_("current-year"), value: "current_year" },
        { text: $_("last-12-months"), value: "last_12_months" },
        { text: $_("custom"), value: "custom" },
    ];

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

    type ActivityChartMode = "bar" | "stacked_bar" | "cumulated_line";
    const activityChartModeItems: SelectItem[] = [
        { text: "chart-bar", value: "bar" },
        { text: "chart-stacked-bar", value: "stacked_bar" },
        { text: "chart-cumulated-line", value: "cumulated_line" },
    ];

    let barChartSelectedValue = $state(barChartSelectItems[0].value);
    let activityChartModes = $state<SelectItem[]>([
        activityChartModeItems[1],
    ]);
    let showActivityChartSettings = $state(false);
    let activityChartSettingsElement = $state<HTMLDivElement>();
    let activityBarMode = $derived<ActivityChartMode | "none">(
        activityChartModes.some((item) => item.value === "stacked_bar")
            ? "stacked_bar"
            : activityChartModes.some((item) => item.value === "bar")
              ? "bar"
              : "none",
    );
    let showCumulativeLine = $derived(
        activityChartModes.some((item) => item.value === "cumulated_line"),
    );

    const categoryColors = [
        "#fb8500",
        "#ffb703",
        "#06d6a0",
        "#219ebc",
        "#8ecae6",
        "#ffafcc",
    ];
    const maxCategoryChartItems = categoryColors.length;

    const conversionFactors = {
        distance: 0.621371,
        elevation_gain: 3.28084,
        elevation_loss: 3.28084,
    };

    function groupedCategoryChartItems(includeSubcategories: boolean) {
        const grouped = new Map<
            string,
            {
                label: string;
                categoryLabel: string;
                subcategoryLabel: string;
                value: number;
                icon: string;
                badgeIcon: string;
                activityKeys: Set<string>;
            }
        >();

        for (const activity of activities) {
            const trail = activity.expand?.trail;
            const category =
                trail?.expand?.category ??
                $categories.find((item) => item.id === trail?.category);
            const activityLabel =
                displayTrailCategoryLabel(activity.expand?.trail, $locale) || "-";
            const activityKey = trailCategoryKey(trail);
            const categoryLabel = displayCategoryName(category, $locale) || "-";
            const subcategoryLabel = includeSubcategories
                ? displaySubcategoryLabel(
                      trail?.expand?.subcategory,
                      $locale,
                  )
                : "";
            const label = includeSubcategories
                ? activityLabel
                : categoryLabel;
            const key = includeSubcategories
                ? `${category?.id ?? trail?.category ?? "-"}:${trail?.expand?.subcategory?.id ?? trail?.subcategory ?? "-"}`
                : category?.id ?? trail?.category ?? "-";
            const item = grouped.get(key);
            if (item) {
                item.value += 1;
                item.activityKeys.add(activityKey);
            } else {
                grouped.set(key, {
                    label,
                    categoryLabel,
                    subcategoryLabel,
                    value: 1,
                    icon: includeSubcategories
                        ? displayTrailCategoryIcon(trail)
                        : displayCategoryIcon(category),
                    badgeIcon: includeSubcategories
                        ? displayTrailCategoryBadgeIcon(trail)
                        : "",
                    activityKeys: new Set([activityKey]),
                });
            }
        }

        return [...grouped.values()].sort((a, b) =>
            a.label.localeCompare(b.label, $locale ?? undefined),
        );
    }

    let categoryChartItems = $derived.by(() => {
        const detailedItems = groupedCategoryChartItems(true);
        let visibleItems = detailedItems;

        if (detailedItems.length > maxCategoryChartItems) {
            const categoryItems = groupedCategoryChartItems(false);
            visibleItems = categoryItems;

            if (categoryItems.length > maxCategoryChartItems) {
                const rankedItems = [...categoryItems].sort(
                    (a, b) =>
                        b.value - a.value ||
                        a.label.localeCompare(b.label, $locale ?? undefined),
                );
                const remainingItems = rankedItems.slice(
                    maxCategoryChartItems - 1,
                );

                visibleItems = [
                    ...rankedItems
                        .slice(0, maxCategoryChartItems - 1)
                        .sort((a, b) =>
                            a.label.localeCompare(
                                b.label,
                                $locale ?? undefined,
                            ),
                        ),
                    {
                        label: `${$_("Other")} (+${remainingItems.length})`,
                        categoryLabel: `${$_("Other")} (+${remainingItems.length})`,
                        subcategoryLabel: "",
                        value: remainingItems.reduce(
                            (sum, item) => sum + item.value,
                            0,
                        ),
                        icon: "fa-ellipsis",
                        badgeIcon: "",
                        activityKeys: new Set(
                            remainingItems.flatMap((item) => [
                                ...item.activityKeys,
                            ]),
                        ),
                    },
                ];
            }
        }

        return visibleItems.map((item, index) => ({
            ...item,
            activityKeys: [...item.activityKeys],
            color: categoryColors[index],
        }));
    });

    let categoryLabels = $derived(categoryChartItems.map((item) => item.label));
    let categoryValues = $derived(categoryChartItems.map((item) => item.value));
    let legendShowsSubcategories = $derived(
        categoryChartItems.some((item) => item.subcategoryLabel),
    );
    let activityCategoryColorMap = $derived(
        Object.fromEntries(
            categoryChartItems.flatMap((item) =>
                item.activityKeys.map((key) => [key, item.color]),
            ),
        ),
    );
    let calendarActivityCategoryColorMap = $derived.by(() => {
        const colorMap = { ...activityCategoryColorMap };
        const usedColors = new Set(Object.values(colorMap));
        const calendarKeys = [
            ...new Set(
                calendarActivities.map((activity) =>
                    trailCategoryKey(activity.expand?.trail),
                ),
            ),
        ].sort();
        let fallbackColorIndex = 0;

        for (const key of calendarKeys) {
            if (colorMap[key]) {
                continue;
            }

            const unusedColor = categoryColors.find(
                (color) => !usedColors.has(color),
            );
            const color =
                unusedColor ??
                categoryColors[fallbackColorIndex++ % categoryColors.length];
            colorMap[key] = color;
            usedColors.add(color);
        }

        return colorMap;
    });

    let categoryChartData = $derived({
        labels: categoryLabels,
        datasets: [
            {
                data: categoryValues,
                backgroundColor: categoryChartItems.map((item) => item.color),
                borderWidth: 0,
                hoverBorderWidth: 0,
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

    function barChartValue(activity: StatisticActivity) {
        let value = activity[
            barChartSelectedValue as
                | "distance"
                | "duration"
                | "elevation_gain"
                | "elevation_loss"
        ] ?? 0;

        if (barChartSelectedValue === "distance") {
            value /= 1000;
        } else if (barChartSelectedValue === "duration") {
            value /= 60 * 60;
        }

        if (page.data.settings?.unit !== "metric") {
            if (barChartSelectedValue === "distance") {
                value *= conversionFactors.distance;
            } else if (barChartSelectedValue === "elevation_gain") {
                value *= conversionFactors.elevation_gain;
            } else if (barChartSelectedValue === "elevation_loss") {
                value *= conversionFactors.elevation_loss;
            }
        }

        return value;
    }

    function formatDateLabel(value: string) {
        return new Date(value).toLocaleDateString(undefined, {
            month: "2-digit",
            day: "2-digit",
            year: "numeric",
            timeZone: "UTC",
        });
    }

    function formatActivityChartDate(
        value: number,
        unit: ActivityChartTimeUnit,
        detailed = false,
    ) {
        const date = new Date(Math.round(value) * millisecondsPerDay);
        if (unit === "day") {
            return date.toLocaleDateString(undefined, {
                month: "2-digit",
                day: "2-digit",
                year: detailed ? "numeric" : undefined,
                timeZone: "UTC",
            });
        }
        if (unit === "month") {
            return date.toLocaleDateString(undefined, {
                month: detailed ? "long" : "short",
                year: detailed ? "numeric" : undefined,
                timeZone: "UTC",
            });
        }

        const year = date.getUTCFullYear();
        if (unit === "quarter") {
            return `Q${Math.floor(date.getUTCMonth() / 3) + 1} ${year}`;
        }
        return String(year);
    }

    type ActivityChartPeriod = {
        start: string;
        end: string;
        dataUnit: ActivityChartTimeUnit;
        tickUnit: ActivityChartTimeUnit;
        dataBuckets: ActivityChartBucket[];
        axisTicks: ActivityChartBucket[];
    };

    let activityChartPeriod = $derived.by((): ActivityChartPeriod => {
        const start = filter.startDate;
        const end = filter.endDate;
        if (!start || !end) {
            return {
                start: "",
                end: "",
                dataUnit: "day",
                tickUnit: "day",
                dataBuckets: [],
                axisTicks: [],
            };
        }

        const dataUnit = activityChartDataUnit(start, end);
        const tickUnit = activityChartTickUnit(start, end);
        return {
            start,
            end,
            dataUnit,
            tickUnit,
            dataBuckets: activityChartBuckets(start, end, dataUnit),
            axisTicks: activityChartBuckets(start, end, tickUnit),
        };
    });

    let activityChartDateAxisRange = $derived.by(() => {
        if (!activityChartPeriod.start || !activityChartPeriod.end) {
            return { min: undefined, max: undefined };
        }

        const min = dateAxisValue(activityChartPeriod.start);
        const max = dateAxisValue(activityChartPeriod.end);
        return min === max
            ? { min: min - 0.49, max: max + 0.49 }
            : { min, max };
    });

    let activityChartSeries = $derived.by(() => {
        const buckets = activityChartPeriod.dataBuckets;
        const bucketIndexByKey = new Map(
            buckets.map((bucket, index) => [bucket.key, index]),
        );
        const itemIndexByActivityKey = new Map<string, number>();
        const valuesByItem = categoryChartItems.map(
            () => buckets.map(() => 0),
        );

        categoryChartItems.forEach((item, index) => {
            item.activityKeys.forEach((key) =>
                itemIndexByActivityKey.set(key, index),
            );
        });

        for (const activity of activities) {
            const itemIndex = itemIndexByActivityKey.get(
                trailCategoryKey(activity.expand?.trail),
            );
            if (itemIndex === undefined) {
                continue;
            }

            const activityDate = dateInputValue(parseDateValue(activity.date));
            const bucketKey = activityChartBucketKey(
                activityDate,
                activityChartPeriod.dataUnit,
            );
            const bucketIndex = bucketIndexByKey.get(bucketKey);
            if (bucketIndex === undefined) {
                continue;
            }
            valuesByItem[itemIndex][bucketIndex] += barChartValue(activity);
        }

        return {
            labels: buckets.map((bucket) => String(bucket.position)),
            valuesByItem,
            totalValues: buckets.map((_, bucketIndex) =>
                valuesByItem.reduce(
                    (sum, values) => sum + values[bucketIndex],
                    0,
                ),
            ),
        };
    });

    let activityChartData = $derived.by(() => {
        const datasets: ChartDataset<"bar" | "line", number[]>[] = [];

        if (activityBarMode === "bar") {
            datasets.push({
                type: "bar",
                label: barChartSelectItems.find(
                    (item) => item.value === barChartSelectedValue,
                )?.text,
                data: activityChartSeries.totalValues,
                backgroundColor: "#3388ff",
                borderRadius: 4,
                borderSkipped: false,
                maxBarThickness: 32,
                order: 1,
                yAxisID: "y",
            });
        } else if (activityBarMode === "stacked_bar") {
            categoryChartItems.forEach((item, index) => {
                datasets.push({
                    type: "bar",
                    label: item.label,
                    data: activityChartSeries.valuesByItem[index],
                    backgroundColor: item.color,
                    borderRadius: 4,
                    borderSkipped: false,
                    maxBarThickness: 32,
                    stack: "activities",
                    order: 1,
                    yAxisID: "y",
                });
            });
        }

        let cumulativeValue = 0;
        if (showCumulativeLine) {
            datasets.push({
                type: "line",
                label: $_("cumulative"),
                data: activityChartSeries.totalValues.map((value) => {
                    cumulativeValue += value;
                    return cumulativeValue;
                }),
                borderColor: "#ef476f",
                backgroundColor: "#ef476f",
                borderWidth: 3,
                fill: false,
                tension: 0.25,
                pointRadius: 0,
                pointHoverRadius: 0,
                order: 0,
                yAxisID: "cumulative",
            });
        }

        return {
            labels: activityChartSeries.labels,
            datasets,
        };
    });

    let totalDistance = $derived(
        activities.reduce((sum, activity) => sum + (activity.distance ?? 0), 0),
    );

    let totalDuration = $derived(
        activities.reduce((sum, activity) => sum + (activity.duration ?? 0), 0),
    );

    let totalElevationGain = $derived(
        activities.reduce(
            (sum, activity) => sum + (activity.elevation_gain ?? 0),
            0,
        ),
    );

    let totalElevationLoss = $derived(
        activities.reduce(
            (sum, activity) => sum + (activity.elevation_loss ?? 0),
            0,
        ),
    );

    let averageSpeed = $derived(
        totalDuration > 0
            ? activities.reduce(
                  (sum, activity) =>
                      sum +
                      (activity.distance && activity.duration
                          ? activity.distance
                          : 0),
                  0,
              ) / totalDuration
            : undefined,
    );

    function handlePeriodChange(value: PeriodMode) {
        if (value === "custom") {
            periodMode = appliedPeriodMode;
            periodRangeModal.openModal();
            return;
        }

        periodMode = value;
        appliedPeriodMode = value;
        const range = datePeriodRange(value);
        filter.startDate = range.start;
        filter.endDate = range.end;
        loadActivities();
    }

    function handleActivityChartModesChange(
        value: SelectItem[],
        changedItem?: SelectItem,
    ) {
        let nextValue = [...value];
        const changedMode = changedItem?.value as ActivityChartMode | undefined;

        if (
            (changedMode === "bar" || changedMode === "stacked_bar") &&
            nextValue.some((item) => item.value === changedMode)
        ) {
            const otherBarMode =
                changedMode === "bar" ? "stacked_bar" : "bar";
            nextValue = nextValue.filter(
                (item) => item.value !== otherBarMode,
            );
        }

        if (!nextValue.length && changedItem) {
            nextValue = [changedItem];
        }

        activityChartModes = nextValue;
    }

    function handleWindowMouseUp(event: MouseEvent) {
        if (
            showActivityChartSettings &&
            !activityChartSettingsElement?.contains(event.target as Node)
        ) {
            showActivityChartSettings = false;
        }
    }

    function handleCustomPeriodApply(range: {
        start: string;
        end: string;
    }) {
        filter.startDate = range.start;
        filter.endDate = range.end;
        periodMode = "custom";
        appliedPeriodMode = "custom";
        loadActivities();
    }

    function resolvedPeriodLabel() {
        if (!filter.startDate || !filter.endDate) {
            return "–";
        }

        return `${formatDateLabel(filter.startDate)} – ${formatDateLabel(filter.endDate)}`;
    }

    function handleCalendarMonthChange(range: { start: string; end: string }) {
        calendarRange = range;
        loadCalendarActivities();
    }

    function showLoadError() {
        show_toast({
            icon: "close",
            text: "Error loading stats.",
            type: "error",
        });
    }

    async function loadActivities() {
        try {
            activities = await profile_stats_index(page.params.handle!, filter);
        } catch (e) {
            showLoadError();
        }
    }

    async function loadCalendarActivities() {
        try {
            calendarActivities = await profile_stats_index(
                page.params.handle!,
                {
                    ...filter,
                    startDate: calendarRange.start,
                    endDate: calendarRange.end,
                },
            );
        } catch (e) {
            showLoadError();
        }
    }

    async function loadAllActivities() {
        try {
            const [nextActivities, nextCalendarActivities] = await Promise.all([
                profile_stats_index(page.params.handle!, filter),
                profile_stats_index(page.params.handle!, {
                    ...filter,
                    startDate: calendarRange.start,
                    endDate: calendarRange.end,
                }),
            ]);
            activities = nextActivities;
            calendarActivities = nextCalendarActivities;
        } catch (e) {
            showLoadError();
        }
    }

    onMount(() => {
        if (
            filter.startDate !== calendarRange.start ||
            filter.endDate !== calendarRange.end
        ) {
            loadCalendarActivities();
        }
    });
</script>

<svelte:head>
    <title>{$_("profile")} | wanderer</title>
</svelte:head>

<svelte:window onmouseup={handleWindowMouseUp} />

<div
    class="grid grid-cols-1 lg:grid-cols-[320px_minmax(0,_1fr)] gap-y-4 max-w-6xl mx-auto"
>
    <div
        class="grid grid-cols-1 lg:grid-cols-[320px_minmax(0,_1fr)] col-span-1 lg:col-span-2 gap-y-4 items-start"
    >
        <div class="min-w-0 lg:mr-4">
            <TrailCategoryFilter
                categories={$categories}
                {filter}
                onupdate={loadAllActivities}
            />
        </div>
        <div
            class="grid grid-cols-[minmax(0,_1fr)_auto_auto] items-end gap-3"
        >
            <Select
                name="statistics-period"
                bind:value={periodMode}
                items={periodSelectItems}
                label={$_("period")}
                fullWidth
                onchange={handlePeriodChange}
            ></Select>
            <span
                class="flex h-10 items-center whitespace-nowrap text-sm text-gray-500"
            >
                {resolvedPeriodLabel()}
            </span>
            <button
                type="button"
                class="btn-icon h-10 w-10 shrink-0"
                aria-label={$_("date-selection")}
                title={$_("date-selection")}
                onclick={() => periodRangeModal.openModal()}
            >
                <i class="fa fa-calendar" aria-hidden="true"></i>
            </button>
        </div>
    </div>
    <div class="space-y-4 grow-0 lg:mr-4">
        <div class="border border-input-border rounded-xl p-6">
            <Calendar
                month={calendarRange.start}
                activities={calendarActivities}
                colorMap={calendarActivityCategoryColorMap}
                onmonthchange={handleCalendarMonthChange}
            ></Calendar>
        </div>
        <div class="border border-input-border rounded-xl p-6">
            <span class="text-gray-500 font-semibold text-lg">
                {$_("categories")}
            </span>
            <div class="pt-6">
                <Pie
                    data={categoryChartData}
                    options={{
                        responsive: true,
                        plugins: {
                            legend: {
                                display: false,
                            },
                        },
                    }}
                />
            </div>
            {#if categoryChartItems.length}
                <ul
                    class="mt-4 flex flex-wrap justify-center gap-x-5 gap-y-2 text-sm text-gray-500"
                >
                    {#each categoryChartItems as item}
                        <li class="flex items-center gap-2">
                            <span
                                class="relative inline-flex w-5 shrink-0 justify-center"
                                style="color: {item.color}"
                            >
                                <i class="fa {item.icon}" aria-hidden="true"></i>
                                {#if item.badgeIcon}
                                    <i
                                        class="fa {item.badgeIcon} absolute -right-0.5 -top-1 text-[8px]"
                                        aria-hidden="true"
                                    ></i>
                                {/if}
                            </span>
                            {#if legendShowsSubcategories}
                                <span>
                                    <span class="text-content">
                                        {item.categoryLabel}
                                    </span>
                                    {#if item.subcategoryLabel}
                                        / {item.subcategoryLabel}
                                    {/if}
                                </span>
                            {:else}
                                <span>{item.label}</span>
                            {/if}
                        </li>
                    {/each}
                </ul>
            {/if}
        </div>
    </div>

    <div
        class="grid grid-cols-1 lg:grid-cols-2 lg:grid-rows-[auto_auto_auto_1fr] gap-4"
    >
        <div
            class="flex flex-col items-center gap-4 border border-input-border rounded-xl p-6"
        >
            <span class="text-gray-500 font-semibold text-lg self-start"
                ><i class="fa fa-hashtag mr-3"></i>{$_("activity", {
                    values: { n: 2 },
                })}</span
            >
            <p class="text-3xl font-bold">{activities.length}</p>
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
                ><i class="fa fa-gauge mr-3"></i>{$_(
                    "average-speed",
                )}</span
            >
            <p class="text-3xl font-bold">{formatSpeed(averageSpeed, 1)}</p>
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
            class="h-full lg:col-span-2 flex flex-col gap-2 border border-input-border rounded-xl p-6"
        >
            <div class="flex items-center justify-between gap-2">
                <span class="text-gray-500 font-semibold text-lg"
                    ><i class="fa fa-calendar mr-3"></i>{$_("activity", {
                        values: { n: 1 },
                    })}</span
                >
                <div
                    class="relative ml-auto"
                    bind:this={activityChartSettingsElement}
                >
                    <button
                        type="button"
                        class="btn-icon"
                        aria-label={`${$_("activity")} ${$_("settings")}`}
                        title={`${$_("activity")} ${$_("settings")}`}
                        aria-expanded={showActivityChartSettings}
                        onclick={() =>
                            (showActivityChartSettings =
                                !showActivityChartSettings)}
                    >
                        <i class="fa fa-wrench" aria-hidden="true"></i>
                    </button>
                    {#if showActivityChartSettings}
                        <div
                            class="absolute right-0 top-full z-20 mt-2 w-72 space-y-3 rounded-xl border border-input-border bg-menu-background p-3 shadow-md"
                        >
                            <MultiSelect
                                label={$_("display")}
                                bind:value={activityChartModes}
                                items={activityChartModeItems}
                                onchange={handleActivityChartModesChange}
                            ></MultiSelect>
                            <Select
                                fullWidth
                                bind:value={barChartSelectedValue}
                                items={barChartSelectItems}
                            ></Select>
                        </div>
                    {/if}
                </div>
            </div>

            <div class="relative min-h-64 lg:min-h-0 flex-1">
                <Bar
                    class="absolute inset-0 h-full w-full"
                    data={activityChartData}
                    options={{
                        maintainAspectRatio: false,
                        interaction: {
                            mode: "index",
                            intersect: false,
                        },
                        plugins: {
                            legend: {
                                display: false,
                            },
                            tooltip: {
                                callbacks: {
                                    title: (items) => {
                                        const value = items[0]?.parsed.x;
                                        return typeof value === "number"
                                            ? formatActivityChartDate(
                                                  value,
                                                  activityChartPeriod.dataUnit,
                                                  true,
                                              )
                                            : "";
                                    },
                                    label: (item) =>
                                        `${item.dataset.label}: ${item.formattedValue} ${barChartUnit()}`,
                                },
                            },
                        },
                        scales: {
                            x: {
                                type: "linear",
                                stacked: activityBarMode === "stacked_bar",
                                min: activityChartDateAxisRange.min,
                                max: activityChartDateAxisRange.max,
                                afterBuildTicks: (axis) => {
                                    axis.ticks =
                                        activityChartPeriod.axisTicks.map(
                                            (bucket) => ({
                                                value: bucket.position,
                                            }),
                                        );
                                },
                                ticks: {
                                    minRotation: 0,
                                    maxRotation: 0,
                                    callback: (value) =>
                                        formatActivityChartDate(
                                            Number(value),
                                            activityChartPeriod.tickUnit,
                                        ),
                                },
                            },
                            y: {
                                display: activityBarMode !== "none",
                                stacked: activityBarMode === "stacked_bar",
                                beginAtZero: true,
                                ticks: {
                                    callback: (value) =>
                                        value + " " + barChartUnit(),
                                },
                            },
                            cumulative: {
                                display:
                                    showCumulativeLine &&
                                    activityBarMode === "none",
                                position: "left",
                                beginAtZero: true,
                                grid: {
                                    drawOnChartArea: false,
                                },
                                ticks: {
                                    callback: (value) =>
                                        value + " " + barChartUnit(),
                                },
                            },
                        },
                    }}
                ></Bar>
            </div>
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
                summitLogs={activities}
                handle={page.params.handle!}
                showCategory
                showTrail
                showRoute
                compactElevationHeaders
                categoryColorMap={activityCategoryColorMap}
            ></SummitLogTable>
        </div>
    </div>
</div>

<DateRangeModal
    bind:this={periodRangeModal}
    start={filter.startDate}
    end={filter.endDate}
    onapply={handleCustomPeriodApply}
></DateRangeModal>
