<script lang="ts">
    import Toggle from "$lib/components/base/toggle.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        onclick: () => void;
        ontoggle: (value: boolean) => void;
        active: boolean;
        toggleDisabled?: boolean;
        settingsDisabled?: boolean;
        img?: string;
        title: string;
        description?: string;
        lastSyncAt?: string;
        error?: string;
    }

    let {
        onclick,
        ontoggle,
        active = $bindable(),
        toggleDisabled = false,
        settingsDisabled = false,
        img,
        title,
        description = "",
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

<div
    class="flex flex-col gap-4 rounded-xl border border-input-border p-4 transition-colors hover:bg-secondary-hover md:flex-row md:items-center md:justify-between"
>
    <div class="flex min-w-0 items-start gap-6">
        {#if img}
            <img
                class="h-16 w-24 shrink-0 object-contain object-left"
                src={img}
                alt="plugin logo"
            />
        {:else}
            <div
                class="flex h-16 w-24 shrink-0 items-center justify-center rounded border border-input-border bg-input-background text-sm font-semibold"
                aria-hidden="true"
            >
                {title.slice(0, 2).toUpperCase()}
            </div>
        {/if}
        <div class="min-w-0">
            <h5 class="truncate text-lg font-semibold">{title}</h5>
            {#if description}
                <p class="line-clamp-2 text-sm text-gray-500">{description}</p>
            {/if}
        </div>
    </div>
    <div class="flex shrink-0 flex-col gap-2 md:items-start">
        <div class="flex items-center justify-between gap-4 md:justify-end">
            <button
                class="btn-secondary"
                class:btn-disabled={settingsDisabled}
                {onclick}
                disabled={settingsDisabled}
                ><i class="fa fa-cogs mr-2"></i>{$_("settings")}</button
            >
            <div class="plugin-card-toggle">
                <Toggle bind:value={active} onchange={ontoggle} disabled={toggleDisabled}></Toggle>
            </div>
        </div>
        <div class="min-h-5 max-w-56 text-xs text-gray-500">
            {#if lastSyncAt}
                <span
                    class:text-red-400={error}
                    class="inline-flex min-w-0 items-center gap-2"
                    title={error
                        ? $_("plugin-setup-error")
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
                <span class="inline-flex items-center gap-2 text-red-400" title={$_("plugin-setup-error")}>
                    <i
                        class="fa fa-triangle-exclamation shrink-0 text-[0.8rem]"
                        aria-hidden="true"
                    ></i>
                    <span class="truncate">{$_("plugin-setup-error")}</span>
                </span>
            {/if}
        </div>
    </div>
</div>
