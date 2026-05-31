<script lang="ts">
    import Toggle from "$lib/components/base/toggle.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        onclick: () => void;
        ontoggle: (value: boolean) => void;
        active: boolean;
        disabled: boolean;
        img?: string;
        title: string;
        description: string;
        lastSyncAt?: string;
        error?: string;
    }

    let {
        onclick,
        ontoggle,
        active = $bindable(),
        disabled,
        img,
        title,
        description,
        lastSyncAt = "",
        error = "",
    }: Props = $props();

    function formatLastSyncAt(value: string) {
        return new Date(value).toLocaleString(undefined, {
            dateStyle: "short",
            timeStyle: "short",
        });
    }
</script>

<div class="flex min-h-[20.5rem] flex-col rounded-lg border border-input-border p-5">
    <div class="mb-5 flex h-24 items-center">
        {#if img}
            <img
                class="max-h-20 max-w-full object-contain object-left"
                src={img}
                alt="plugin logo"
            />
        {:else}
            <div
                class="flex h-20 w-20 items-center justify-center rounded border border-input-border bg-input-background text-lg font-semibold"
                aria-hidden="true"
            >
                {title.slice(0, 2).toUpperCase()}
            </div>
        {/if}
    </div>
    <div class="flex-1">
        <h5 class="text-xl font-semibold">{title}</h5>
        <p class="text-sm text-gray-500">
            {description}
        </p>
    </div>
    <div class="mt-4 min-h-5 text-xs text-gray-500">
        {#if lastSyncAt}
            <span
                class:text-red-400={error}
                class="inline-flex min-w-0 items-center gap-2"
                title={error
                    ? error
                    : `${$_("last-sync")}: ${formatLastSyncAt(lastSyncAt)}`}
            >
                {#if error}
                    <i
                        class="fa fa-triangle-exclamation shrink-0 text-[0.8rem]"
                        aria-hidden="true"
                    ></i>
                {:else}
                    <i
                        class="fa fa-clock shrink-0 text-[0.8rem]"
                        aria-hidden="true"
                    ></i>
                {/if}
                <span class="truncate">{formatLastSyncAt(lastSyncAt)}</span>
            </span>
        {:else if error}
            <span class="inline-flex items-center gap-2 text-red-400" title={error}>
                <i
                    class="fa fa-triangle-exclamation shrink-0 text-[0.8rem]"
                    aria-hidden="true"
                ></i>
                <span>Sync</span>
            </span>
        {/if}
    </div>
    <div class="mt-5 flex items-center justify-between gap-4">
        <button class="btn-secondary" {onclick}
            ><i class="fa fa-cogs mr-2"></i>{$_("settings")}</button
        >
        <Toggle bind:value={active} onchange={ontoggle} {disabled}></Toggle>
    </div>
</div>
