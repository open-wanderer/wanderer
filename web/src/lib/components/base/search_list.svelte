<script lang="ts">
    import { isURL } from "#lib/util/file_util.ts";
    import type { Snippet } from "svelte";
    import type { SearchItem } from "./search.svelte";
    import { _ } from "svelte-i18n";
    interface Props {
        id?: string;
        items?: SearchItem[];
        element?: HTMLUListElement;
        onclick?: (e: Event, item: SearchItem) => void;
        prepend?: Snippet<[any]>;
        extraClasses?: string;
    }

    let {
        id,
        items = [],
        element = $bindable(),
        onclick,
        prepend,
        extraClasses,
    }: Props = $props();
</script>

<ul
    {id}
    class="menu absolute bg-menu-background border border-input-border rounded-xl shadow-md overflow-x-hidden overflow-y-scroll max-h-72 {extraClasses}"
    style="z-index: 1001"
>
    {#if items.length == 0}
        <li
            class="menu-item flex items-center px-4 py-3 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors"
        >
            {$_("no-results")}
        </li>
    {/if}
    {#each items as item}
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <li
            class="menu-item flex items-center px-4 py-3 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors"
            tabindex="0"
            onmousedown={(e) => { if (e.button === 0) onclick?.(e, item); }}
            onkeydown={(e) => {
                if (e.key !== "Enter" && e.key !== " ") return;
                e.preventDefault();
                onclick?.(e, item);
            }}
        >
            {#if prepend}{@render prepend({ item })}
            {:else if isURL(item.icon)}
                <img
                    class="rounded-full w-8 mr-4 aspect-square"
                    src={item.icon}
                    alt="avatar"
                />
            {:else}
                <i class="fa fa-{item.icon} basis-8 shrink-0"></i>
            {/if}

            <div>
                <p>{item.text}</p>
                {#if item.description}
                    <p class="text-sm text-gray-500">
                        {item.description}
                    </p>
                {/if}
            </div>
        </li>
    {/each}
</ul>
