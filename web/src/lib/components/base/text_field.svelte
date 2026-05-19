<script lang="ts">
    import { _ } from "svelte-i18n";
    import type { ChangeEventHandler, FocusEventHandler, FormEventHandler } from "svelte/elements";

    interface Props {
        name?: string;
        value?: string | number;
        placeholder?: string;
        disabled?: boolean;
        label?: string;
        error?: string | string[] | null;
        icon?: string;
        extraClasses?: string;
        type?: "text" | "password" | "search";
        autocomplete?: "on" | "off";
        onchange?: ChangeEventHandler<HTMLInputElement>;
        oninput?: FormEventHandler<HTMLInputElement>;
        onfocusin?: FocusEventHandler<HTMLInputElement>;
        onfocusout?: FocusEventHandler<HTMLInputElement>;
    }

    let {
        name = "",
        value = $bindable(""),
        placeholder = "",
        disabled = false,
        label = "",
        error = "",
        icon = "",
        extraClasses = "",
        type = "text",
        autocomplete = "on",
        onchange,
        oninput,
        onfocusin,
        onfocusout
    }: Props = $props();
    function typeAction(node: HTMLInputElement) {
        node.type = type;
    }
</script>

<div>
    {#if label.length}
        <label for={name} class="text-sm font-medium pb-1">
            {label}
        </label>
    {/if}
    <div class="flex items-center gap-2">
        {#if icon.length > 0}
            <div class="w-6">
                <i class="fa fa-{icon}"></i>
            </div>
        {/if}
        <input
            {name}
            class="bg-input-background border border-input-border rounded-md p-3 transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0 w-full {extraClasses}"
            class:border-red-400={(error?.length ?? 0) > 0}
            class:bg-input-background-error={(error?.length ?? 0) > 0}
            class:text-gray-500={disabled}
            {disabled}
            {autocomplete}
            use:typeAction
            bind:value
            {onchange}
            {oninput}
            {onfocusin}
            {onfocusout}
            {placeholder}
        />
    </div>

    {#if error}
        <span class="textfield-error text-xs text-red-400">
            {error instanceof Array ? $_(error[0]) : error}
        </span>
    {/if}
</div>

<style>
    :global(input[type="search"]::-webkit-search-cancel-button) {
        -webkit-appearance: none;
        appearance: none;
    }
</style>
