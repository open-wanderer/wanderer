<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        title?: string;
        text: string;
        action?: string;
        deny?: string;
        id?: string;
        onconfirm?: () => void
        oncancel?: () => void
    }

    let {
        title = $_("confirm-deletion"),
        text,
        action = "delete",
        deny ="cancel",
        id = "confirm-modal",
        onconfirm,
        oncancel
    }: Props = $props();

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    function cancel() {
        oncancel?.();
        modal.closeModal!();
    }
    
    function confirm() {
        onconfirm?.()
        modal.closeModal!();
    }
</script>

<Modal {id} {title} bind:this={modal}>
    {#snippet content()}
        <p>{text}</p>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={cancel}
                >{$_(deny)}</button
            >
            <button
                id="confirm"
                class={action === "delete" ? "btn-danger" : "btn-primary"}
                type="button"
                onclick={confirm}
                name="delete">{$_(action)}</button
            >
        </div>
    {/snippet}</Modal
>
