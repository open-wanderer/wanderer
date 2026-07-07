<script module lang="ts">
    export type TrailBulkEditChanges = {
        category?: {
            category: string;
            subcategory: string;
        };
        difficulty?: "easy" | "moderate" | "difficult";
    };
</script>

<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import Select from "$lib/components/base/select.svelte";
    import CategoryPicker, {
        type CategoryPickerSelection,
    } from "./category_picker.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        selectedCount?: number;
        initialCategorySelection?: CategoryPickerSelection;
        initialDifficulty?: TrailBulkEditChanges["difficulty"];
        onapply?: (changes: TrailBulkEditChanges) => Promise<void> | void;
    }

    let {
        selectedCount = 0,
        initialCategorySelection,
        initialDifficulty,
        onapply,
    }: Props = $props();

    let modal: Modal;
    let applyCategory = $state(false);
    let applyDifficulty = $state(false);
    let categorySelection: CategoryPickerSelection | undefined = $state();
    let categoryValue = $state("");
    let difficultyValue: "easy" | "moderate" | "difficult" = $state("easy");
    let loading = $state(false);

    let canApply = $derived(
        (applyCategory && categorySelection !== undefined) || applyDifficulty,
    );

    export function openModal() {
        applyCategory = false;
        applyDifficulty = false;
        categorySelection = initialCategorySelection;
        categoryValue = categoryValueFromSelection(initialCategorySelection);
        difficultyValue = initialDifficulty ?? "easy";
        modal.openModal();
    }

    function categoryValueFromSelection(selection?: CategoryPickerSelection) {
        if (!selection) {
            return "";
        }

        return selection.subcategory
            ? `subcategory:${selection.subcategory}`
            : `category:${selection.category}`;
    }

    function handleCategoryChange(selection: CategoryPickerSelection) {
        categorySelection = selection;
        categoryValue = selection.subcategory
            ? `subcategory:${selection.subcategory}`
            : `category:${selection.category}`;
    }

    async function apply(closeModal: () => void) {
        if (!canApply) {
            return;
        }

        const changes: TrailBulkEditChanges = {};
        if (applyCategory) {
            if (!categorySelection) {
                return;
            }
            changes.category = categorySelection;
        }
        if (applyDifficulty) {
            changes.difficulty = difficultyValue;
        }

        loading = true;
        try {
            await onapply?.(changes);
            closeModal();
        } finally {
            loading = false;
        }
    }
</script>

<Modal
    id="trail-bulk-edit-modal"
    title={$_("adjust")}
    size="md:min-w-sm"
    bind:this={modal}
>
    {#snippet content()}
        <p class="text-sm text-gray-500">
            {$_("bulk-edit-selected-trails", {
                values: { n: selectedCount },
            })}
        </p>

        <div class="space-y-4">
            <label class="flex items-center gap-2">
                <input
                    type="checkbox"
                    class="h-4 w-4 accent-primary"
                    bind:checked={applyCategory}
                />
                <span class="font-medium">{$_("category")}</span>
            </label>
            {#if applyCategory}
                <CategoryPicker
                    value={categoryValue}
                    label=""
                    loadData
                    fixedDropdown
                    onchange={handleCategoryChange}
                ></CategoryPicker>
            {/if}

            <label class="flex items-center gap-2">
                <input
                    type="checkbox"
                    class="h-4 w-4 accent-primary"
                    bind:checked={applyDifficulty}
                />
                <span class="font-medium">{$_("difficulty")}</span>
            </label>
            {#if applyDifficulty}
                <Select
                    value={difficultyValue}
                    onchange={(value) => (difficultyValue = value)}
                    items={[
                        { text: $_("easy"), value: "easy" },
                        { text: $_("moderate"), value: "moderate" },
                        { text: $_("difficult"), value: "difficult" },
                    ]}
                ></Select>
            {/if}
        </div>
    {/snippet}

    {#snippet footer({ closeModal })}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" type="button" onclick={closeModal}>
                {$_("cancel")}
            </button>
            <button
                class="btn-primary"
                class:btn-disabled={!canApply || loading}
                disabled={!canApply || loading}
                type="button"
                onclick={() => apply(closeModal)}
            >
                {$_("apply")}
            </button>
        </div>
    {/snippet}
</Modal>
