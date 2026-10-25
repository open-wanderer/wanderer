import type { ComboboxItem } from "$lib/components/base/combobox.svelte";

// Suggestions the combobox should offer. In multiple mode, items that are
// already selected are left out (matched by text, like the component does when
// adding an item). In single mode all items are returned.
export function visibleComboboxItems(
    items: ComboboxItem[],
    value: string | ComboboxItem[] | undefined,
    multiple: boolean,
): ComboboxItem[] {
    if (!multiple || !Array.isArray(value)) {
        return items;
    }
    const selected = new Set(value.map((i) => i.text));
    return items.filter((i) => !selected.has(i.text));
}
