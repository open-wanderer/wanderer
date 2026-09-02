<script lang="ts">
    import { dateInputValue } from "$lib/util/date_util";
    import { _ } from "svelte-i18n";
    import Calendar from "./calendar.svelte";
    import Modal from "./modal.svelte";

    interface Props {
        start?: string;
        end?: string;
        onapply?: (range: { start: string; end: string }) => void;
    }

    let { start, end, onapply }: Props = $props();

    let modal: Modal;
    let draftStart = $state<string>();
    let draftEnd = $state<string>();
    let visibleMonth = $state(dateInputValue(new Date()));
    let waitingForEnd = $state(false);

    export function openModal() {
        draftStart = start;
        draftEnd = end;
        waitingForEnd = false;
        visibleMonth = start ?? dateInputValue(new Date());
        modal.openModal();
    }

    function selectDate(selectedDate: Date) {
        const value = dateInputValue(selectedDate);

        if (!waitingForEnd) {
            draftStart = value;
            draftEnd = undefined;
            waitingForEnd = true;
            return;
        }

        if (draftStart && value < draftStart) {
            draftEnd = draftStart;
            draftStart = value;
        } else {
            draftEnd = value;
        }
        waitingForEnd = false;
    }

    function formatDate(value?: string) {
        if (!value) {
            return "–";
        }

        return new Date(value).toLocaleDateString(undefined, {
            month: "2-digit",
            day: "2-digit",
            year: "numeric",
            timeZone: "UTC",
        });
    }

    function apply(closeModal: () => void) {
        if (!draftStart || !draftEnd || waitingForEnd) {
            return;
        }

        onapply?.({ start: draftStart, end: draftEnd });
        closeModal();
    }
</script>

<Modal
    id="statistics-date-range-modal"
    title={$_("date-selection")}
    size="w-[calc(100vw-2rem)] max-w-md"
    bind:this={modal}
>
    {#snippet content()}
        <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
                <div class="mb-1 text-gray-500">{$_("after")}</div>
                <div class="font-semibold">{formatDate(draftStart)}</div>
            </div>
            <div>
                <div class="mb-1 text-gray-500">{$_("before")}</div>
                <div class="font-semibold">{formatDate(draftEnd)}</div>
            </div>
        </div>
        <Calendar
            month={visibleMonth}
            selectedStart={draftStart}
            selectedEnd={draftEnd}
            onclick={selectDate}
            onmonthchange={(range) => (visibleMonth = range.start)}
        ></Calendar>
    {/snippet}

    {#snippet footer({ closeModal })}
        <div class="flex items-center justify-end gap-4">
            <button class="btn-secondary" type="button" onclick={closeModal}>
                {$_("cancel")}
            </button>
            <button
                class="btn-primary"
                class:btn-disabled={!draftStart || !draftEnd || waitingForEnd}
                disabled={!draftStart || !draftEnd || waitingForEnd}
                type="button"
                onclick={() => apply(closeModal)}
            >
                {$_("apply")}
            </button>
        </div>
    {/snippet}
</Modal>
