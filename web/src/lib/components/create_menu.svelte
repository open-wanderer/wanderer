<script lang="ts">
    import { tick } from "svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        inline?: boolean;
        onselect?: () => void;
        onurlimport: () => void;
    }
    let { inline = false, onselect, onurlimport }: Props = $props();
    const id = $props.id();
    let open = $state(false);
    let root: HTMLDivElement;
    let trigger: HTMLButtonElement;
    let panel = $state<HTMLElement>();

    function close(restoreFocus = false) {
        open = false;
        if (restoreFocus) trigger.focus();
    }
    async function toggle(focusFirst = false) {
        open = !open;
        if (open && focusFirst) {
            await tick();
            panel?.querySelector<HTMLElement>("a, button")?.focus();
        }
    }
    function selected() {
        close();
        onselect?.();
    }
    function importUrl() {
        selected();
        onurlimport();
    }
    function outside(event: PointerEvent) {
        if (open && event.target instanceof Node && !root.contains(event.target)) close();
    }
    function focusOut(event: FocusEvent) {
        if (open && event.relatedTarget instanceof Node && !root.contains(event.relatedTarget)) close();
    }
    async function keydown(event: KeyboardEvent) {
        if (!(event.target instanceof Node) || !root?.contains(event.target)) return;
        if (event.key === "Escape" && open) {
            event.preventDefault(); event.stopPropagation(); close(true);
        } else if (event.key === "ArrowDown" && event.target === trigger) {
            event.preventDefault();
            if (!open) await toggle(true);
            else panel?.querySelector<HTMLElement>("a, button")?.focus();
        } else if (open && panel?.contains(event.target)) {
            const items = Array.from(panel.querySelectorAll<HTMLElement>("a, button"));
            const current = items.indexOf(event.target as HTMLElement);
            let next: number;
            if (event.key === "ArrowDown") next = (current + 1) % items.length;
            else if (event.key === "ArrowUp") next = (current - 1 + items.length) % items.length;
            else if (event.key === "Home") next = 0;
            else if (event.key === "End") next = items.length - 1;
            else return;
            event.preventDefault();
            items[next]?.focus();
        }
    }
</script>

<svelte:window onpointerdown={outside} onkeydown={keydown} />

<div class="create-menu" class:inline bind:this={root} onfocusout={focusOut}>
    <button type="button" class="btn-primary btn-large create-trigger" bind:this={trigger}
        aria-label={$_("create")} aria-expanded={open} aria-controls={`${id}-panel`} onclick={() => toggle()}>
        <i class="fa fa-plus" aria-hidden="true"></i><span>{$_("create")}</span>
        <i class="fa fa-caret-down" aria-hidden="true"></i>
    </button>
    {#if open}
        <nav id={`${id}-panel`} class="create-panel" aria-label={$_("create")} bind:this={panel}>
            <a href="/trail/edit/new" data-sveltekit-reload onclick={selected}>
                <i class="fa fa-route" aria-hidden="true"></i><span>{$_("new-trail")}</span>
            </a>
            <a href="/lists/edit/new" onclick={selected}>
                <i class="fa fa-list-ul" aria-hidden="true"></i><span>{$_("new-list")}</span>
            </a>
            <div class="import-heading">{$_("import")}</div>
            <a href="/settings/export" onclick={selected}>
                <i class="fa fa-file-import" aria-hidden="true"></i><span>{$_("from-file")}</span>
            </a>
            <button type="button" onclick={importUrl}>
                <i class="fa fa-link" aria-hidden="true"></i><span>{$_("from-url")}</span>
            </button>
        </nav>
    {/if}
</div>

<style>
    .create-menu { position: relative; }
    .create-trigger { align-items: center; cursor: pointer; display: flex; gap: .6rem; justify-content: center; }
    .create-trigger > i:last-child { margin-left: .5rem; }
    .create-panel { background: rgba(var(--menu-background)); border: 1px solid rgba(var(--input-border)); border-radius: .85rem; box-shadow: 0 8px 24px rgb(0 0 0 / .12); color: rgba(var(--content)); max-width: calc(100vw - 2rem); padding: .45rem; position: absolute; right: 0; top: calc(100% + .5rem); min-width: 100%; width: max-content; z-index: 60; }
    .create-panel a, .create-panel button { align-items: center; border-radius: .45rem; cursor: pointer; display: flex; font: inherit; font-size: .85rem; gap: .75rem; min-height: 2.6rem; padding: .55rem .7rem; text-align: left; width: 100%; }
    .create-panel a:hover, .create-panel button:hover { background: rgba(var(--secondary-hover)); }
    .create-panel i { color: rgba(var(--content), .6); text-align: center; width: 1.1rem; }
    .create-panel a:focus-visible, .create-panel button:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: -1px; }
    .import-heading { border-top: 1px solid rgba(var(--separator)); color: rgba(var(--content), .6); font-size: .7rem; margin-top: .4rem; padding: .7rem .7rem .25rem; }
    .inline { margin: 0 1rem; }
    .inline .create-trigger { width: 100%; }
    .inline .create-panel { box-shadow: none; margin-top: .5rem; position: static; width: 100%; }
</style>
