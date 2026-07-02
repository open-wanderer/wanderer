<script lang="ts">
    import type { Snippet } from "svelte";

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

    export function openModal() {
        document.body.style.top = `-${window.scrollY}px`;
        document.body.style.position = "fixed";
        document.body.style.width = "100%";

        (document.getElementById(id) as HTMLDialogElement).showModal();
    }

    export function closeModal() {
        const scrollY = document.body.style.top;
        document.body.style.position = "";
        document.body.style.top = "";
        window.scrollTo(0, parseInt(scrollY || "0") * -1);

        (document.getElementById(id) as HTMLDialogElement).close();
    }
</script>

{@render children?.({ openModal })}
<dialog
    {id}
    tabindex="-1"
    aria-hidden="true"
    onclose={() => onclose?.()}
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
