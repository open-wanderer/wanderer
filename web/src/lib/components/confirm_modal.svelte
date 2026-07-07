<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import type { Snippet } from "svelte";
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
        children?: Snippet
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
        onalternative,
        children,
    }: Props = $props();

    let modal: Modal;
    let closingFromAction = false;

    export function openModal() {
        closingFromAction = false;
        modal.openModal();
    }

    function handleModalClose() {
        if (closingFromAction) {
            closingFromAction = false;
            return;
        }
        oncancel?.();
    }

    function cancel() {
        closingFromAction = true;
        modal.closeModal!();
        oncancel?.();
    }

    function alternativeAction() {
        closingFromAction = true;
        modal.closeModal!();
        onalternative?.();
    }
    
    function confirm() {
        closingFromAction = true;
        modal.closeModal!();
        onconfirm?.()
    }
</script>

<Modal {id} {title} bind:this={modal} onclose={handleModalClose}>
    {#snippet content()}
        {#if children}
            {@render children()}
        {:else}
            <p>{text}</p>
        {/if}
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
