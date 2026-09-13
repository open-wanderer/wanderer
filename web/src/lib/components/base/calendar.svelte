<script lang="ts">
	import type { StatisticActivity } from "$lib/models/statistic_activity";
	import { range } from "$lib/util/array_util";
	import { trailCategoryKey } from "$lib/util/category_util";
	import {
		dateInputValue,
		isSameDay,
		isToday,
		monthDateRange,
		parseDateValue,
	} from "../../util/date_util";
	import { untrack } from "svelte";
	import { _, date, locale } from "svelte-i18n";
	interface Props {
		activities?: StatisticActivity[];
		colorMap?: Record<string, string>;
		month?: string;
		minMonth?: string;
		maxMonth?: string;
		selectedStart?: string;
		selectedEnd?: string;
		onmonthchange?: (data: { start: string; end: string }) => void;
		onclick?: (date: Date) => void;
	}
	type CalendarDay = {
		date: Date;
		today: boolean;
		activities: StatisticActivity[];
	};

	let {
		activities = [],
		colorMap = {},
		month,
		minMonth,
		maxMonth,
		selectedStart,
		selectedEnd,
		onmonthchange,
		onclick,
	}: Props = $props();

	const initialMonth = untrack(() => month ? parseDateValue(month) : new Date());
	let currentMonth = $state(initialMonth.getMonth());
	let currentYear = $state(initialMonth.getFullYear());
	let currentMonthArray: (CalendarDay | null)[] = $derived(
		generateMonthArray(currentYear, currentMonth, activities),
	);
	let currentMonthValue = $derived(
		dateInputValue(new Date(currentYear, currentMonth, 1)),
	);
	let minimumMonthValue = $derived(
		minMonth ? monthDateRange(parseDateValue(minMonth)).start : undefined,
	);
	let maximumMonthValue = $derived(
		maxMonth ? monthDateRange(parseDateValue(maxMonth)).start : undefined,
	);
	let canGoToPreviousMonth = $derived(
		!minimumMonthValue || currentMonthValue > minimumMonthValue,
	);
	let canGoToNextMonth = $derived(
		!maximumMonthValue || currentMonthValue < maximumMonthValue,
	);

	$effect(() => {
		if (!month) {
			return;
		}
		const selectedMonth = parseDateValue(month);
		currentYear = selectedMonth.getFullYear();
		currentMonth = selectedMonth.getMonth();
	});

	function calculateFirstDayOfMonthDayOfWeek(year: number, month: number) {
		const date = new Date(year, month, 1);
		const day = date.getDay();

		return day == 0 ? 6 : day - 1;
	}

	function daysInMonth(year: number, month: number) {
		const days = new Date(year, month + 1, 0).getDate();
		return days;
	}

	function generateMonthArray(
		year: number,
		month: number,
		activities: StatisticActivity[],
	) {
		const a: (CalendarDay | null)[] = [];
		const firstDay = calculateFirstDayOfMonthDayOfWeek(year, month);
		const totalDays = daysInMonth(year, month);

		for (let i = 0; i < 42; i++) {
			if (i < firstDay || i - firstDay >= totalDays) {
				a.push(null);
			} else {
				const date = new Date(year, month, i + 1 - firstDay);
				const today = isToday(date);

				const activitiesAtDate = activities.filter((activity) =>
					isSameDay(date, parseDateValue(activity.date)),
				);

				a.push({
					date,
					today,
					activities: activitiesAtDate,
				});
			}
		}

		return a;
	}

	function activityColors(day: CalendarDay | null): string[] {
		return [
			...new Set(
				(day?.activities ?? [])
					.map((activity) =>
						colorMap[trailCategoryKey(activity.expand?.trail)],
					)
					.filter((color): color is string => Boolean(color)),
			),
		];
	}

	function calendarDateValue(day: CalendarDay | null): string {
		return day ? dateInputValue(day.date) : "";
	}

	function isSelected(day: CalendarDay | null): boolean {
		const value = calendarDateValue(day);
		return Boolean(
			value &&
				selectedStart &&
				selectedEnd &&
				value >= selectedStart &&
				value <= selectedEnd,
		);
	}

	function isRangeBoundary(day: CalendarDay | null): boolean {
		const value = calendarDateValue(day);
		return Boolean(
			value && (value === selectedStart || value === selectedEnd),
		);
	}

	function calendarDayLabel(
		day: CalendarDay,
		includeActivityCount = false,
	): string {
		const dateLabel = day.date.toLocaleDateString($locale ?? undefined, {
			weekday: "long",
			year: "numeric",
			month: "long",
			day: "numeric",
		});

		if (!includeActivityCount) {
			return dateLabel;
		}

		const activityCount = day.activities.length;
		return `${dateLabel}: ${activityCount} ${$_("activity", {
			values: { n: activityCount },
		})}`;
	}

	function monthPlus() {
		if (!canGoToNextMonth) {
			return;
		}
		if (currentMonth === 11) {
			currentYear += 1;
			currentMonth = 0;
		} else {
			currentMonth += 1;
		}
		onmonthchange?.(monthDateRange(new Date(currentYear, currentMonth, 1)));
	}

	function monthMinus() {
		if (!canGoToPreviousMonth) {
			return;
		}
		if (currentMonth === 0) {
			currentYear -= 1;
			currentMonth = 11;
		} else {
			currentMonth -= 1;
		}
		onmonthchange?.(monthDateRange(new Date(currentYear, currentMonth, 1)));
	}
</script>

<div class="calendar-header mb-6 flex w-full items-center justify-between">
	<div class="calendar-month-year basis-full">
		<span class="text-lg">{$date(new Date(currentYear, currentMonth, 1, 0, 0), { format: 'monthName' } )}</span>
		<span>{currentYear}</span>
	</div>
	<button
		type="button"
		aria-label="Previous month"
		class="btn-icon mr-2 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent"
		disabled={!canGoToPreviousMonth}
		onclick={monthMinus}><i class="fa fa-caret-left" aria-hidden="true"></i></button
	>
	<button
		type="button"
		aria-label="Next month"
		class="btn-icon disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent"
		disabled={!canGoToNextMonth}
		onclick={monthPlus}><i class="fa fa-caret-right" aria-hidden="true"></i></button
	>
</div>
<div class="calendar-body">
	<div class="grid grid-cols-7">
		{#each range(7), i}
			<div
				class="calendar-weekday flex items-center justify-center h-10 text-gray-500"
			>
				{$_("calendar.weekdays." + i)}
			</div>
		{/each}
	</div>
	<div
		class="grid grid-cols-7 grid-rows-6 gap-1"
		style="aspect-ratio: 1.17/1"
	>
		{#each { length: 42 } as _, i}
			{@const day = currentMonthArray[i]}
			{@const colors = activityColors(currentMonthArray[i])}
			{#if day && onclick}
				<button
					type="button"
					aria-label={calendarDayLabel(day, true)}
					aria-current={day.today ? "date" : undefined}
					aria-pressed={isSelected(day)}
					class="calendar-day relative flex cursor-pointer items-center justify-center rounded-xl"
					class:today={day.today}
					class:has-activities={day.activities.length > 0}
					class:range-selected={isSelected(day)}
					class:range-boundary={isRangeBoundary(day)}
					onclick={() => onclick?.(day.date)}
				>
					<span aria-hidden="true">{day.date.getDate()}</span>
					{#if colors.length}
						<span
							class="absolute bottom-1 left-1/2 flex max-w-full -translate-x-1/2 flex-wrap justify-center gap-0.5"
							aria-hidden="true"
						>
							{#each colors as color}
								<span
									class="h-1 w-1 rounded-full"
									style="background-color: {color}"
								></span>
							{/each}
						</span>
					{/if}
				</button>
			{:else if day}
				<time
					datetime={calendarDateValue(day)}
					aria-current={day.today ? "date" : undefined}
					class="calendar-day relative flex items-center justify-center rounded-xl"
					class:today={day.today}
					class:has-activities={day.activities.length > 0}
				>
					<span aria-hidden="true">{day.date.getDate()}</span>
					<span class="sr-only">{calendarDayLabel(day, true)}</span>
					{#if colors.length}
						<span
							class="absolute bottom-1 left-1/2 flex max-w-full -translate-x-1/2 flex-wrap justify-center gap-0.5"
							aria-hidden="true"
						>
							{#each colors as color}
								<span
									class="h-1 w-1 rounded-full"
									style="background-color: {color}"
								></span>
							{/each}
						</span>
					{/if}
				</time>
			{:else}
				<div class="calendar-day" aria-hidden="true"></div>
			{/if}
		{/each}
	</div>
</div>

<style lang="postcss">
	@reference "tailwindcss";
	@reference "../../../css/app.css";

	.calendar-weekday {
		font-weight: 600;
	}
	.calendar-month-year span {
		font-weight: 600;
	}
	.calendar-day.today {
		@apply border border-input-border;
	}
	.calendar-day.has-activities {
		@apply bg-input-background;
	}
	.calendar-day.range-selected {
		@apply bg-primary/20;
	}
	.calendar-day.range-boundary {
		@apply bg-primary text-white;
	}
</style>
