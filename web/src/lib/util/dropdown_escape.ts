// Only consume Escape owned by this menu; fields and dialogs elsewhere keep it.
export function closeDropdownOnEscape(event: KeyboardEvent, root: HTMLElement | undefined, close: () => void) {
    if (!root || event.key !== "Escape" || event.isComposing || event.defaultPrevented) return;
    const focused = document.activeElement;
    if (focused !== document.body && !root.contains(focused)) return;
    event.preventDefault();
    event.stopPropagation();
    close();
    const trigger = root.querySelector<HTMLElement>(".dropdown-toggle button, .dropdown-toggle a[href]");
    if (trigger?.getClientRects().length) trigger.focus({ preventScroll: true });
}
