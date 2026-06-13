<script lang="ts">
	import type { SummitLog } from "$lib/models/summit_log";
    import { range } from "$lib/util/array_util";
	import { displayCategoryName } from "$lib/util/category_util";
	import { isSameDay, isToday } from "../../util/date_util";
	import { _, date, locale } from "svelte-i18n";
	interface Props {
		logs?: SummitLog[];
		colorMap?: Record<string, string>;
		onforward?: (data: { start: Date; end: Date }) => void;
		onbackward?: (data: { start: Date; end: Date }) => void;
		onclick?: (date: Date) => void;
	}

	let {
		logs = [],
		colorMap = {},
		onforward,
		onbackward,
		onclick,
	}: Props = $props();

	const today = new Date();
	let currentMonth = $state(today.getMonth());
	let currentYear = $state(today.getFullYear());
	let currentMonthArray: ({
		date: Date | undefined;
		today: boolean;
		log?: SummitLog;
	} | null)[] = $derived(generateMonthArray(currentYear, currentMonth, logs));

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
		logs: SummitLog[],
	) {
		const a: ({ date: Date; today: boolean; log?: SummitLog } | null)[] =
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

				const logAtDate = logs.find((l) =>
					isSameDay(date, new Date(l.date)),
				);

				a.push({ date: date, today: today, log: logAtDate });
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
		onforward?.({
			start: new Date(currentYear, currentMonth, 1),
			end: new Date(currentYear, currentMonth + 1, 0),
		});
	}

	function monthMinus() {
		if (currentMonth == 0) {
			currentYear--;
			currentMonth = 11;
		} else {
			currentMonth--;
		}
		onbackward?.({
			start: new Date(currentYear, currentMonth, 1),
			end: new Date(currentYear, currentMonth + 1, 0),
		});
	}

	function colorKey(a: typeof currentMonthArray, i: number) {
		return displayCategoryName(
			a[i]?.log?.expand?.trail?.expand?.category,
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
