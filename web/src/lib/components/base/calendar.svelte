<script lang="ts">
	import type { StatisticActivity } from "$lib/models/statistic_activity";
    import { range } from "$lib/util/array_util";
	import { displayTrailCategoryLabel } from "$lib/util/category_util";
	import {
		isSameDay,
		isToday,
		monthDateRange,
		parseDateValue,
	} from "../../util/date_util";
	import { _, date, locale } from "svelte-i18n";
	interface Props {
		activities?: StatisticActivity[];
		colorMap?: Record<string, string>;
		month?: string;
		onmonthchange?: (data: { start: string; end: string }) => void;
		onclick?: (date: Date) => void;
	}

	let {
		activities = [],
		colorMap = {},
		month,
		onmonthchange,
		onclick,
	}: Props = $props();

	const today = new Date();
	let currentMonth = $state(today.getMonth());
	let currentYear = $state(today.getFullYear());
	let currentMonthArray: ({
		date: Date | undefined;
		today: boolean;
		activity?: StatisticActivity;
	} | null)[] = $derived(generateMonthArray(currentYear, currentMonth, activities));

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
		const a: ({ date: Date; today: boolean; activity?: StatisticActivity } | null)[] =
			[];
		const firstDay = calculateFirstDayOfMonthDayOfWeek(year, month);
		const totalDays = daysInMonth(year, month);

		for (let i = 0; i < 42; i++) {
			if (i < firstDay || i - firstDay >= totalDays) {
				a.push(null);
			} else {
				const date = new Date(
					currentYear,
					currentMonth,
					i + 1 - firstDay,
				);
				const today = isToday(date);

				const activityAtDate = activities.find((activity) =>
					isSameDay(date, parseDateValue(activity.date)),
				);

				a.push({ date: date, today: today, activity: activityAtDate });
			}
		}

		return a;
	}

	function monthPlus() {
		if (currentMonth == 11) {
			currentYear++;
			currentMonth = 0;
		} else {
			currentMonth++;
		}
		onmonthchange?.(monthDateRange(new Date(currentYear, currentMonth, 1)));
	}

	function monthMinus() {
		if (currentMonth == 0) {
			currentYear--;
			currentMonth = 11;
		} else {
			currentMonth--;
		}
		onmonthchange?.(monthDateRange(new Date(currentYear, currentMonth, 1)));
	}

	function colorKey(a: typeof currentMonthArray, i: number) {
		return displayTrailCategoryLabel(
			a[i]?.activity?.expand?.trail,
			$locale,
		);
	}

	function handleDateClick(date?: Date) {
		if (!date) {
			return;
		}
		onclick?.(date);
	}
</script>

<div class="calendar-header w-full flex items-center justify-between mb-6">
	<div class="calendar-month-year basis-full">
		<span class="text-lg">{$date(new Date(currentYear, currentMonth, 1, 0, 0), { format: 'monthName' } )}</span>
		<span>{currentYear}</span>
	</div>
	<button
		aria-label="Previous month"
		class="btn-icon mr-2"
		onclick={monthMinus}><i class="fa fa-caret-left"></i></button
	>
	<button aria-label="Next month" class="btn-icon" onclick={monthPlus}
		><i class="fa fa-caret-right"></i></button
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
			<button
				class="calendar-day flex items-center justify-center rounded-xl"
				onclick={() => handleDateClick(currentMonthArray[i]?.date)}
				class:today={currentMonthArray[i]?.today}
				style="background-color: {colorMap[
					colorKey(currentMonthArray, i)
				] ?? ''}"
			>
				{currentMonthArray[i]?.date?.getDate() ?? ""}
			</button>
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
</style>
