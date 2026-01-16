<script lang="ts">
    import { _ } from "svelte-i18n";
    import type { ChangeEventHandler } from "svelte/elements";
    
    interface Props {
        name?: string;
        value?: string | Date;
        label?: string;
        error?: string | string[] | null;
        onchange?: ChangeEventHandler<HTMLInputElement>;
    }

    let {
        name = "",
        value = $bindable(),
        label = "",
        error = "",
        onchange
    }: Props = $props();
</script>

<div>
    {#if label.length}
        <p class="text-sm font-medium pb-1">
            {label}
        </p>
    {/if}
    <div class="flex items-center gap-2">
        <input
            {name}
            class="bg-input-background border border-input-border rounded-md p-3 transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0 w-full dark:[color-scheme:dark]"
            class:border-red-400={(error?.length ?? 0) > 0}
            class:bg-input-background-error={(error?.length ?? 0) > 0}
            type="date"
            bind:value
            {onchange}
        />
    </div>

    {#if error}
        <span class="textfield-error text-xs text-red-400">
            {error instanceof Array ? $_(error[0]) : error}
        </span>
    {/if}
</div>
