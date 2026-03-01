<script module lang="ts">
    export type DropdownItem = {
        text: string;
        value: any;
        icon?: string;
    };
</script>

<script lang="ts">
    import { tick, type Snippet } from "svelte";
    import { fly } from "svelte/transition";

    interface Props {
        items?: DropdownItem[];
        size?: string;
        children?: Snippet<[any]>;
        onchange?: (item: DropdownItem) => void;
    }

    let { items = [], size = "regular", children, onchange }: Props = $props();

    let isOpen = $state(false);

    let dropdownElement: HTMLUListElement | undefined = $state();
    let dropdownToggleElement: HTMLDivElement;

    export async function toggleMenu(e: MouseEvent) {
        e.stopPropagation();
        e.preventDefault();

        isOpen = !isOpen;

        if (isOpen && dropdownElement) {
            await tick();

            const toggleRect = dropdownToggleElement.getBoundingClientRect();
            const dropdownRect = dropdownElement.getBoundingClientRect();
            dropdownElement.style.visibility = "";

            // Viewport dimensions
            const viewportHeight = window.innerHeight;
            const spaceBelowViewport = viewportHeight - toggleRect.bottom;
            const spaceAboveViewport = toggleRect.top;

            // Container dimensions
            const scrollContainer =
                dropdownElement.closest(".scroll-x-only") ?? document.body;
            const containerRect = scrollContainer.getBoundingClientRect();
            const spaceBelowContainer =
                containerRect.bottom - toggleRect.bottom;
            const spaceAboveContainer = toggleRect.top - containerRect.top;

            // Effective space
            const spaceBelow = Math.min(
                spaceBelowViewport,
                spaceBelowContainer,
            );
            const spaceAbove = Math.min(
                spaceAboveViewport,
                spaceAboveContainer,
            );

            // Determine direction
            const openUpward =
                spaceBelow < dropdownRect.height &&
                spaceAbove > dropdownRect.height;

            if (openUpward) {
                dropdownElement.classList.remove("rounded-b-xl");
                dropdownElement.classList.add("rounded-t-xl");
                dropdownElement.style.top = `${-8 - dropdownRect.height}px`;
            } else {
                dropdownElement.classList.remove("rounded-t-xl");
                dropdownElement.classList.add("rounded-b-xl");
                dropdownElement.style.top = `${toggleRect.height + 8}px`;
            }
        }
    }

    function closeMenu() {
        isOpen = false;
    }

    function handleItemClick(
        e: MouseEvent,
        item: { text: string; value: any },
    ) {
        e.preventDefault();
        e.stopPropagation();
        onchange?.(item);
        closeMenu();
    }

    function handleWindowClick(e: MouseEvent) {
        if (
            (e.target as HTMLElement).parentElement?.classList.contains(
                "dropdown-toggle",
            )
        ) {
            return;
        }

        isOpen = false;
    }
</script>

<svelte:window onmouseup={handleWindowClick} />

<div class="dropdown relative">
    <div class="dropdown-toggle" bind:this={dropdownToggleElement}>
        {#if children}{@render children({ toggleMenu })}{:else}
            <button
                aria-label="Toggle menu"
                class="btn-icon flex items-center justify-center"
                onclick={toggleMenu}
                type="button"
            >
                <i class="fa fa-ellipsis-vertical text-{size}"></i>
            </button>
        {/if}
    </div>

    {#if isOpen}
        <ul
            class="menu absolute bg-menu-background border border-input-border shadow-md rounded-l-xl rounded-b-xl right-0 overflow-hidden"
            class:none={isOpen}
            style="z-index: 1001"
            in:fly={{ y: -10, duration: 150 }}
            out:fly={{ y: -10, duration: 150 }}
            bind:this={dropdownElement}
        >
            {#each items as item}
                <li
                    class="menu-item flex items-center px-4 py-3 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors"
                    role="presentation"
                    onclick={(e) => handleItemClick(e, item)}
                >
                    {#if item.icon}
                        <i class="fa fa-{item.icon} mr-3"></i>
                    {/if}
                    <span class="whitespace-nowrap">{item.text}</span>
                </li>
            {/each}
        </ul>
    {/if}
</div>

<style>
</style>
