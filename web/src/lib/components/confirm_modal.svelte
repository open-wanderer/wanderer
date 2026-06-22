<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        title?: string;
        text: string;
        action?: string;
        deny?: string;
        alternative?: string;
        id?: string;
        onconfirm?: () => void
        oncancel?: () => void
        onalternative?: () => void
    }

    let {
        title = $_("confirm-deletion"),
        text,
        action = "delete",
        deny ="cancel",
        alternative,
        id = "confirm-modal",
        onconfirm,
        oncancel,
        onalternative
    }: Props = $props();

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    function cancel() {
        modal.closeModal!();
        oncancel?.();
    }

    function alternativeAction() {
        modal.closeModal!();
        onalternative?.();
    }
    
    function confirm() {
        modal.closeModal!();
        onconfirm?.()
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
            {#if alternative}
                <button class="btn-secondary" type="button" onclick={alternativeAction}
                    >{$_(alternative)}</button
                >
            {/if}
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
