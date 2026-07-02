<script module lang="ts">
    export type SingleSelectItem = {
        text: string;
        value: any;
    };
</script>

<script lang="ts">
    import { onMount, tick } from "svelte";

    interface Props {
        name?: string;
        items?: SingleSelectItem[];
        value?: any;
        label?: string;
        ariaLabel?: string;
        placeholder?: string;
        disabled?: boolean;
        onchange?: (value: any) => void;
    }

    let {
        name = "",
        items = [],
        value = $bindable(""),
        label = "",
        ariaLabel = "",
        placeholder = "",
        disabled = false,
        onchange,
    }: Props = $props();

    let open = $state(false);
    let trigger: HTMLButtonElement | undefined = $state();
    let menu: HTMLUListElement | undefined = $state();
    let menuStyle = $state("");
    let activeIndex = $state(-1);

    let selectedItem = $derived(items.find((item) => item.value === value));
    let selectedIndex = $derived(items.findIndex((item) => item.value === value));

    async function positionMenu() {
        await tick();
        if (!trigger) return;

        const rect = trigger.getBoundingClientRect();
        const gap = 4;
        const viewportPadding = 12;
        const maxHeight = 256;
        const spaceBelow = window.innerHeight - rect.bottom - viewportPadding;
        const spaceAbove = rect.top - viewportPadding;
        const openAbove = spaceBelow < 160 && spaceAbove > spaceBelow;
        const availableHeight = Math.max(
            80,
            Math.min(maxHeight, openAbove ? spaceAbove - gap : spaceBelow - gap),
        );
        const top = openAbove ? rect.top - gap - availableHeight : rect.bottom + gap;

        menuStyle = [
            `left: ${rect.left}px`,
            `top: ${top}px`,
            `width: ${rect.width}px`,
            `max-height: ${availableHeight}px`,
        ].join("; ");
    }

    async function scrollActiveItemIntoView() {
        await tick();
        menu?.querySelector<HTMLElement>(`[data-single-select-index="${activeIndex}"]`)
            ?.scrollIntoView({ block: "nearest" });
    }

    function selectItem(item: SingleSelectItem) {
        if (disabled) return;
        value = item.value;
        open = false;
        onchange?.(item.value);
    }

    async function openMenu(index = selectedIndex >= 0 ? selectedIndex : 0) {
        if (disabled) return;
        activeIndex = items.length ? Math.max(0, Math.min(index, items.length - 1)) : -1;
        open = true;
        await positionMenu();
        await scrollActiveItemIntoView();
    }

    function closeMenu() {
        open = false;
        activeIndex = -1;
    }

    async function toggle() {
        if (disabled) return;
        if (open) {
            closeMenu();
        } else {
            await openMenu();
        }
    }

    async function setActiveIndex(index: number) {
        if (!items.length) return;
        activeIndex = (index + items.length) % items.length;
        await scrollActiveItemIntoView();
    }

    async function handleKeydown(event: KeyboardEvent) {
        if (disabled) return;

        switch (event.key) {
            case "ArrowDown":
                event.preventDefault();
                if (!open) {
                    await openMenu(selectedIndex >= 0 ? selectedIndex : 0);
                } else {
                    await setActiveIndex(activeIndex + 1);
                }
                break;
            case "ArrowUp":
                event.preventDefault();
                if (!open) {
                    await openMenu(selectedIndex >= 0 ? selectedIndex : items.length - 1);
                } else {
                    await setActiveIndex(activeIndex - 1);
                }
                break;
            case "Home":
                if (!open) return;
                event.preventDefault();
                await setActiveIndex(0);
                break;
            case "End":
                if (!open) return;
                event.preventDefault();
                await setActiveIndex(items.length - 1);
                break;
            case "Enter":
            case " ":
                event.preventDefault();
                if (!open) {
                    await openMenu();
                } else if (activeIndex >= 0 && items[activeIndex]) {
                    selectItem(items[activeIndex]);
                }
                break;
            case "Escape":
                if (!open) return;
                event.preventDefault();
                closeMenu();
                break;
        }
    }

    onMount(() => {
        function handleDocumentMouseDown(event: MouseEvent) {
            const target = event.target as Node | null;
            if (!target || trigger?.contains(target) || menu?.contains(target)) {
                return;
            }
            open = false;
        }

        function handleDocumentKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") {
                closeMenu();
            }
        }

        function handleDocumentScroll(event: Event) {
            if (!open) return;
            const target = event.target as Node | null;
            if (target && menu?.contains(target)) {
                return;
            }
            closeMenu();
        }

        function handleWindowResize() {
            if (!open) return;
            positionMenu();
        }

        document.addEventListener("mousedown", handleDocumentMouseDown);
        document.addEventListener("keydown", handleDocumentKeyDown);
        document.addEventListener("scroll", handleDocumentScroll, true);
        window.addEventListener("resize", handleWindowResize);

        return () => {
            document.removeEventListener("mousedown", handleDocumentMouseDown);
            document.removeEventListener("keydown", handleDocumentKeyDown);
            document.removeEventListener("scroll", handleDocumentScroll, true);
            window.removeEventListener("resize", handleWindowResize);
        };
    });
</script>

<div class="relative">
    {#if label.length}
        <label for={name} class="text-sm font-medium pb-1">
            {label}
        </label>
    {/if}
    <button
        bind:this={trigger}
        id={name}
        type="button"
        aria-label={ariaLabel || label || name}
        aria-haspopup="listbox"
        aria-expanded={open}
        class="relative flex h-10 w-full items-center rounded-md border border-input-border bg-input-background px-3 text-left transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0"
        class:opacity-75={disabled}
        class:cursor-default={disabled}
        disabled={disabled}
        onclick={toggle}
        onkeydown={handleKeydown}
    >
        {#if selectedItem}
            <span class="truncate">{selectedItem.text}</span>
        {:else}
            <span class="truncate text-gray-400">{placeholder}</span>
        {/if}
        <i
            class="fa fa-caret-down absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 transition-transform"
            class:rotate-180={open}
        ></i>
    </button>

    {#if open}
        <ul
            bind:this={menu}
            role="listbox"
            class="fixed z-[1001] overflow-y-auto rounded-md border border-input-border bg-menu-background shadow-lg"
            style={menuStyle}
        >
            {#each items as item, i}
                <li
                    role="option"
                    aria-selected={item.value === value}
                    tabindex="-1"
                    class="flex cursor-pointer items-center justify-between px-3 py-2 hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus"
                    class:bg-menu-item-background-focus={activeIndex === i}
                    data-single-select-index={i}
                    onmousedown={(event) => {
                        event.preventDefault();
                        selectItem(item);
                    }}
                >
                    <span class="truncate">{item.text}</span>
                    {#if item.value === value}
                        <i class="fa fa-check ml-3"></i>
                    {/if}
                </li>
            {/each}
        </ul>
    {/if}
</div>
