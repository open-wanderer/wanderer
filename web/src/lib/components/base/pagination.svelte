<script lang="ts">
    import { range } from "$lib/util/array_util";

    interface Props {
        page: number;
        totalPages: number;
        perPage: number;
        onpagination?: (page: number, perPage: number) => void;
    }

    let { page, totalPages, perPage, onpagination }: Props = $props();

    const maxPagesToRender = 6;

    function update(clickedPage: number) {
        if (page !== clickedPage) {
            onpagination?.(clickedPage, perPage);
        }
    }

    let numberGenerator = $derived(() => {
        if (totalPages < 4) {
            return [];
        }
        if (totalPages <= maxPagesToRender) {
            return range(totalPages - 2, 2);
        }
        if (page < maxPagesToRender - 2) {
            return range(maxPagesToRender - 1, 2);
        }
        if (page > totalPages - 3) {
            return range(totalPages - 2, totalPages - 5);
        }
        if (page > 2 && page <= totalPages - 3) {
            return range(page + Math.max(1, maxPagesToRender - 5), page - 2);
        }
        return [];
    });
</script>

<div class="flex gap-x-4 justify-center">
    {#if totalPages > 1}
        <button
            aria-label="Next page"
            class:text-gray-500={page == 1}
            disabled={page == 1}
            onclick={() => update(page - 1)}
            ><i class="fa fa-caret-left"></i></button
        >
        <button
            class="w-8 aspect-square rounded-lg transition-colors border-input-border hover:bg-menu-item-background-hover"
            class:border={page == 1}
            onclick={() => update(1)}>1</button
        >
        {#if totalPages > maxPagesToRender && page >= maxPagesToRender - 2 && page > 2}
            <span>...</span>
        {:else if totalPages > 2}
            <button
                class="w-8 aspect-square rounded-lg transition-colors border-input-border hover:bg-menu-item-background-hover"
                class:border={page == 2}
                onclick={() => update(2)}>2</button
            >
        {/if}
        {#each numberGenerator() as i}
            <button
                class="w-8 aspect-square rounded-lg transition-colors border-input-border hover:bg-menu-item-background-hover"
                class:border={page == i + 1}
                onclick={() => update(i + 1)}>{i + 1}</button
            >
        {/each}
        {#if totalPages > maxPagesToRender && page < totalPages - 3}
            <span>...</span>
        {:else if totalPages > 3}
            <button
                class="w-8 aspect-square rounded-lg transition-colors border-input-border hover:bg-menu-item-background-hover"
                class:border={page == totalPages - 1}
                onclick={() => update(totalPages - 1)}>{totalPages - 1}</button
            >
        {/if}
        <button
            class="w-8 aspect-square rounded-lg transition-colors border-input-border hover:bg-menu-item-background-hover"
            class:border={page == totalPages}
            onclick={() => update(totalPages)}>{totalPages}</button
        >
        <button
            aria-label="Previous page"
            class:text-gray-500={page == totalPages}
            disabled={page == totalPages}
            onclick={() => update(page + 1)}
            ><i class="fa fa-caret-right"></i></button
        >
    {/if}
</div>
