<script module lang="ts">
    export type ComboboxItem = {
        text: string;
        value: any;
        icon?: string;
    };
</script>

<script lang="ts">
    import type { ChangeEventHandler } from "svelte/elements";
    import Chip from "./chip.svelte";
    import { textMatchSegments } from "#lib/util/text_match.ts";
    import { visibleComboboxItems } from "#lib/util/combobox_util.ts";

    interface Props {
        name?: string;
        icon?: string;
        label?: string;
        value?: string | ComboboxItem[];
        items?: ComboboxItem[];
        multiple?: boolean;
        chips?: boolean;
        dismissOnEscape?: boolean;
        placeholder?: string;
        extraClasses?: string;
        onchange?: ChangeEventHandler<HTMLInputElement>;
        onupdate?: (q: string) => void;
        normalizeNewItemText?: (text: string) => string | null;
    }

    let {
        name = "",
        icon = "",
        label = "",
        multiple = false,
        value = $bindable(multiple ? [] : undefined),
        items = [],
        chips = false,
        dismissOnEscape = false,
        placeholder = "",
        extraClasses = "",
        onchange,
        onupdate,
        normalizeNewItemText,
    }: Props = $props();

    let wrapper: HTMLDivElement;
    function dismissOutside(event: Event) {
        if (dismissOnEscape && dropDownOpen && event.target instanceof Node && !wrapper.contains(event.target)) searching = false;
    }
    function leaveInput(event: FocusEvent) {
        if (!dismissOnEscape || !(event.relatedTarget instanceof Element) || !wrapper.querySelector('ul')?.contains(event.relatedTarget)) searching = false;
    }

    let searching: boolean = $state(false);

    let inputValue: string = $state("");

    let visibleItems = $derived(visibleComboboxItems(items, value, multiple));

    let dropDownOpen = $derived(
        ((multiple && inputValue.length > 0) ||
            (!multiple && (value as string)?.length > 0)) &&
            visibleItems.length > 0 &&
            searching,
    );

    let relevantValue = $derived(multiple ? inputValue : ((value as string) ?? ""));

    function handleEscape(event: KeyboardEvent) {
        if (dismissOnEscape && event.key === "ArrowDown" && event.target instanceof HTMLInputElement) {
            searching = true;
        }
        if (!dismissOnEscape || !dropDownOpen || event.key !== "Escape" || event.isComposing) return;
        event.preventDefault();
        event.stopPropagation();
        (event.currentTarget as HTMLElement).querySelector<HTMLInputElement>('input')?.focus();
        searching = false;
    }

    async function onSearchType() {
        if (dismissOnEscape) searching = true;
        const relevantValue = multiple ? inputValue : (value as string);

        update(relevantValue);
    }

    function update(q: string) {
        onupdate?.(q);
    }

    function handleItemClick(e: Event, item: ComboboxItem) {
        e.stopPropagation();

        if (multiple) {
            if ((value as ComboboxItem[]).some((i) => i.text == item.text)) {
                return;
            }
            inputValue = "";
            value = [...(value as ComboboxItem[]), item];
        } else {
            value = item.value;
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (!multiple) {
            return;
        }

        if (e.key == "Enter") {
            e.preventDefault();
            e.stopPropagation();
            if (!inputValue.length) {
                return;
            }
            let matchingItemFromSuggestions = items.find(
                (i) => i.text == inputValue,
            );
            const newItemText = matchingItemFromSuggestions
                ? matchingItemFromSuggestions.text
                : normalizeNewItemText
                  ? normalizeNewItemText(inputValue)
                  : inputValue;
            if (newItemText === null || (value as ComboboxItem[]).some((i) => i.text == newItemText)) {
                inputValue = "";
                return;
            }
            if (!matchingItemFromSuggestions && normalizeNewItemText) {
                matchingItemFromSuggestions = items.find((i) => i.text == newItemText);
            }

            value = [
                ...(value as ComboboxItem[]),
                matchingItemFromSuggestions
                    ? matchingItemFromSuggestions
                    : {
                          text: newItemText,
                          value: null,
                      },
            ];

            inputValue = "";
        } else if (e.key == "Backspace" && inputValue.length == 0) {
            value = (value as ComboboxItem[]).filter(
                (_, i) => i !== value.length - 1,
            );
        }
    }

    function getInputValue() {
        if (multiple) {
            return inputValue;
        } else {
            return value as string;
        }
    }

    function setInputValue(v: string) {
        if (multiple) {
            inputValue = v;
        } else {
            value = v;
        }
    }
</script>

<svelte:document onfocusin={dismissOutside} onpointerdown={dismissOutside} />

<div bind:this={wrapper} class="relative {extraClasses}" onkeydowncapture={handleEscape} role="presentation"
    onfocusout={event => {
        if (!(event.relatedTarget instanceof Node) || !event.currentTarget.contains(event.relatedTarget)) searching = false;
    }} onclick={event => { if (dismissOnEscape && event.target instanceof HTMLInputElement) searching = true; }}>
    {#if label.length}
        <label for={name} class="text-sm font-medium pb-1">
            {label}
        </label>
    {/if}
    <div
        class="flex items-center gap-1 flex-wrap bg-input-background border border-input-border rounded-md p-3 transition-colors focus-within:border-input-border-focus focus-within:ring-0 w-full {extraClasses}"
    >
        {#if multiple}
            {#each value as ComboboxItem[] as v, i}
                {#if chips}
                    <Chip
                        text={v.text}
                        closable
                        onclick={(e) => {
                            if (dismissOnEscape) {
                                searching = false;
                                wrapper.querySelector('input')?.focus();
                                searching = false;
                            }
                            value = (value as ComboboxItem[]).filter(
                                (_, idx) => i != idx,
                            );
                        }}
                    ></Chip>
                {:else}
                    <span
                        >{v.text}{i < (value as ComboboxItem[]).length - 1
                            ? ","
                            : ""}</span
                    >
                {/if}
            {/each}
        {/if}
        <input
            class="flex-1 min-w-24 bg-input-background focus:outline-none"
            type="search"
            {name}
            oninput={onSearchType}
            autocomplete="off"
            {onchange}
            placeholder={value.length ? undefined : placeholder}
            onfocusin={() => (searching = true)}
            onfocusout={leaveInput}
            onkeydown={(e) => handleKeydown(e)}
            bind:value={getInputValue, setInputValue}
        />
    </div>

    {#if dropDownOpen}
        <ul
            class="menu absolute bg-menu-background border border-input-border rounded-xl shadow-md w-full max-h-64 overflow-x-hidden overflow-y-scroll"
            class:none={!dropDownOpen}
            style="z-index: 1001"
        >
            {#each visibleItems as item}
                <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
                <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
                <li
                    class="menu-item flex items-center px-4 py-3 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors"
                    tabindex="0"
                    onmousedown={(e) => { if (e.button === 0) handleItemClick(e, item); }}
                    onkeydown={(e) => {
                        if (e.key !== "Enter" && e.key !== " ") return;
                        e.preventDefault();
                        handleItemClick(e, item);
                    }}
                >
                    {#if item.icon}
                        <i class="fa fa-{item.icon} mr-6"></i>
                    {/if}
                    <p class="text-ellipsis">
                        {#each textMatchSegments(item.text, relevantValue) as segment}
                            {#if segment.matched}<strong>{segment.text}</strong>{:else}{segment.text}{/if}
                        {/each}
                    </p>
                </li>
            {/each}
        </ul>
    {/if}
</div>
