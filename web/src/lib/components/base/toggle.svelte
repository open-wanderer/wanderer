<script lang="ts">
    interface Props {
        name?: string;
        icon?: string;
        value?: boolean;
        label?: string;
        error?: string;
        disabled?: boolean;
        onchange?: (value: boolean) => void;
    }

    let {
        name = "",
        icon = "",
        value = $bindable(false),
        label = "",
        error = "",
        disabled = false,
        onchange,
    }: Props = $props();

    function handleToggleChange() {
        onchange?.(value);
    }
</script>

<div>
    <label
        class="relative my-2 inline-flex items-center"
        class:cursor-pointer={!disabled}
    >
        <input
            {name}
            bind:checked={value}
            type="checkbox"
            class="sr-only peer"
            value="1"
            {disabled}
            onchange={handleToggleChange}
        />
        <div
            class="w-11 h-6 bg-input-background border border-input-border peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-input-ring rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-toggle-active"
        ></div>
        {#if label}
            <span class="ms-3 text-sm font-medium">{label}</span>
            {#if icon.length}
                <i class="fa fa-{icon} ml-2"></i>
            {/if}
        {/if}
    </label>

    <p class="toggle-error text-xs text-red-400">
        {error}
    </p>
</div>
