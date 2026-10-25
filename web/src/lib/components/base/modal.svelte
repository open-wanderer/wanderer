<script module lang="ts">
    const scrollLocks = new Set<HTMLDialogElement>();
    let scrollState: {
        body: HTMLElement;
        x: number;
        y: number;
        styles: { property: string; value: string; priority: string }[];
    } | undefined;

    function acquireScrollLock(dialog: HTMLDialogElement) {
        if (scrollLocks.size === 0) {
            const body = document.body;
            scrollState = {
                body,
                x: window.scrollX,
                y: window.scrollY,
                styles: ["position", "top", "width"].map((property) => ({
                    property,
                    value: body.style.getPropertyValue(property),
                    priority: body.style.getPropertyPriority(property),
                })),
            };
            body.style.top = `-${scrollState.y}px`;
            body.style.position = "fixed";
            body.style.width = "100%";
        }
        scrollLocks.add(dialog);
    }

    function releaseScrollLock(dialog: HTMLDialogElement | undefined) {
        if (!dialog || !scrollLocks.delete(dialog) || scrollLocks.size > 0 || !scrollState) {
            return;
        }
        const saved = scrollState;
        scrollState = undefined;
        for (const { property, value, priority } of saved.styles) {
            saved.body.style.setProperty(property, value, priority);
        }
        window.scrollTo({ left: saved.x, top: saved.y, behavior: "instant" });
    }
</script>

<script lang="ts">
    import { onDestroy, type Snippet } from "svelte";

    interface Props {
        id: string;
        title: string;
        size?: string;
        children?: Snippet<[any]>;
        content?: Snippet;
        footer?: Snippet<[any]>;
        onclose?: () => void;
    }

    let {
        id,
        title,
        size = "max-w-2xl",
        children,
        content,
        footer,
        onclose,
    }: Props = $props();

    let dialog: HTMLDialogElement | undefined;
    let lockedDialog: HTMLDialogElement | undefined;

    export function openModal() {
        if (!dialog || dialog.open) return;
        acquireScrollLock(dialog);
        lockedDialog = dialog;
        try {
            dialog.showModal();
        } catch (error) {
            releaseScrollLock(dialog);
            throw error;
        }
    }

    export function closeModal() {
        dialog?.close();
        releaseScrollLock(dialog);
    }

    function handleClose(event: Event) {
        const closedDialog = event.currentTarget as HTMLDialogElement;
        // A queued close event can arrive after this dialog has been reopened.
        if (!closedDialog.open) releaseScrollLock(closedDialog);
        onclose?.();
    }

    onDestroy(() => releaseScrollLock(lockedDialog));
</script>

{@render children?.({ openModal })}
<dialog
    {id}
    bind:this={dialog}
    tabindex="-1"
    aria-hidden="true"
    onclose={handleClose}
    class="{size} max-h-full rounded-xl text-content"
>
    <!-- Modal content -->
    <div class="bg-background shadow rounded-xl">
        <!-- Modal header -->
        <div
            class="flex items-center justify-between p-4 md:p-5 border-b border-separator rounded-t"
        >
            <h3 class="text-xl font-semibold">
                {title}
            </h3>
            <button
                type="button"
                class="rounded-full btn-icon"
                onclick={closeModal}
            >
                <i class="fa fa-close"></i>
                <span class="sr-only">Close modal</span>
            </button>
        </div>
        <!-- Modal body -->
        <div class="p-4 md:p-5 space-y-4">
            {@render content?.()}
        </div>
        <!-- Modal footer -->
        <div class="p-4 md:p-5 border-t border-separator rounded-b">
            {@render footer?.({ closeModal })}
        </div>
    </div>
</dialog>

<style lang="postcss">
    @reference "tailwindcss";
    dialog::backdrop {
        @apply bg-gray-500 opacity-50;
    }
</style>
