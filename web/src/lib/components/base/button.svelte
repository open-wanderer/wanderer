<script lang="ts">
    import type { Snippet } from "svelte";
    import type { MouseEventHandler } from "svelte/elements";

    interface Props {
        type?: "button" | "submit" | "reset" | null | undefined;
        icon?: string;
        extraClasses?: string;
        primary?: boolean;
        secondary?: boolean;
        large?: boolean;
        loading?: boolean;
        disabled?: boolean;
        tooltip?: string | undefined;
        form?: string;
        id?: string | undefined;
        "aria-label"?: string;
        "aria-busy"?: boolean;
        children?: Snippet;
        onclick?: MouseEventHandler<HTMLButtonElement>;
    }

    let {
        type = undefined,
        icon = "",
        extraClasses = "",
        primary = false,
        secondary = false,
        large = false,
        loading = false,
        disabled = false,
        tooltip = undefined,
        id = undefined,
        "aria-label": ariaLabel = undefined,
        "aria-busy": ariaBusy = undefined,
        form,
        children,
        onclick = undefined,
    }: Props = $props();
</script>

<button
    class="relative {extraClasses} flex items-center justify-center"
    class:btn-primary={primary}
    class:btn-secondary={secondary}
    class:btn-large={large}
    class:btn-disabled={disabled || loading}
    disabled={disabled || loading}
    class:tooltip={(tooltip?.length ?? 0) > 0}
    data-title={tooltip}
    {form}
    {onclick}
    {type}
    {id}
    aria-label={ariaLabel}
    aria-busy={ariaBusy}
>
    <div class:invisible={loading}>
        {#if icon}
            <i class="fa fa-{icon} mr-2"></i>
        {/if}
        {@render children?.()}
    </div>
    {#if loading}
        <div class="absolute aspect-square spinner"></div>
    {/if}
</button>
