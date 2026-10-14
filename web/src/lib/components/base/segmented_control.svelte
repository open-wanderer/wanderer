<script module lang="ts">
    export type SegmentedOption<T = any> = {
        value: T;
        label: string;
        icon?: string;
        disabled?: boolean;
    };
</script>

<script lang="ts" generics="T = any">
    interface Props {
        name?: string;
        label?: string;
        options: SegmentedOption<T>[];
        value?: T;
        disabled?: boolean;
        extraClasses?: string;
        onchange?: (value: T) => void;
    }

    let {
        name = "",
        label = "",
        options = [],
        value = $bindable(),
        disabled = false,
        extraClasses = "",
        onchange,
    }: Props = $props();

    let hiddenInputEl: HTMLInputElement | undefined = $state();

    function select(optionValue: T) {
        if (disabled) return;
        value = optionValue;
        if (hiddenInputEl) {
            if (typeof optionValue === "boolean") {
                hiddenInputEl.checked = optionValue;
            } else {
                hiddenInputEl.value = String(optionValue ?? "");
            }
            hiddenInputEl.dispatchEvent(new Event("input", { bubbles: true }));
            hiddenInputEl.dispatchEvent(new Event("change", { bubbles: true }));
        }
        onchange?.(optionValue);
    }
</script>

<div class="my-2 {extraClasses}">
    {#if label}
        <span class="block text-sm font-medium pb-1.5 text-content">
            {label}
        </span>
    {/if}

    {#if name}
        {#if typeof value === "boolean"}
            <input
                bind:this={hiddenInputEl}
                type="checkbox"
                {name}
                value="1"
                class="sr-only"
                checked={value}
                {disabled}
                tabindex="-1"
                aria-hidden="true"
                onchange={(e) => {
                    value = e.currentTarget.checked as unknown as T;
                    onchange?.(value);
                }}
            />
        {:else}
            <input
                bind:this={hiddenInputEl}
                type="hidden"
                {name}
                value={String(value ?? "")}
                {disabled}
                onchange={(e) => {
                    value = e.currentTarget.value as unknown as T;
                    onchange?.(value);
                }}
            />
        {/if}
    {/if}

    <div
        class="inline-flex p-1 bg-input-background border border-input-border rounded-xl gap-1 items-center select-none"
        role="radiogroup"
        aria-label={label || name}
    >
        {#each options as option}
            {@const isSelected = value === option.value}
            <button
                type="button"
                role="radio"
                aria-checked={isSelected}
                disabled={disabled || option.disabled}
                onclick={() => select(option.value)}
                class="flex items-center justify-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-input-ring {isSelected
                    ? 'bg-background text-content shadow-xs border border-input-border'
                    : 'text-gray-500 hover:text-content hover:bg-menu-item-background-hover border border-transparent'}"
                class:cursor-pointer={!disabled && !option.disabled}
                class:opacity-50={disabled || option.disabled}
            >
                {#if option.icon}
                    <i class="fa fa-{option.icon} text-sm"></i>
                {/if}
                <span>{option.label}</span>
            </button>
        {/each}
    </div>
</div>
