<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";
    import TextField from "../base/text_field.svelte";

    interface Props {
        token: string | null;
    }

    let { token = $bindable() }: Props = $props();

    let modal: Modal;
    let copied: boolean = $state(false);

    export function openModal() {
        copied = false;
        modal.openModal();
    }

    export function closeModal() {
        token = null;
        modal.closeModal();
    }

    function copyTokenToClipboard() {
        if (!token) {
            return;
        }
        navigator.clipboard.writeText(token);
        copied = true;
    }
</script>

<Modal
    id="api-token-success-modal"
    size="md:min-w-md"
    title={$_("new-token-generated")}
    bind:this={modal}
>
    {#snippet content()}
        <p class="max-w-lg">
            Please copy your new API token now. For your security, we cannot
            show it to you again. If you lose this token, you will need to
            delete it and generate a new one.
        </p>
        <div class="flex items-center gap-x-4 flex-nowrap">
            <div class="flex-1">
                <TextField value={token ?? ""} disabled></TextField>
            </div>
            <button
                class="btn-secondary"
                onclick={() => copyTokenToClipboard()}
                aria-label="copy token to clipboard"
                ><i class="fa {copied ? 'fa-check' : 'fa-clipboard'}"
                ></i></button
            >
        </div>
    {/snippet}
    {#snippet footer()}
        <button class="btn-primary" onclick={() => modal.closeModal()}
            >{$_("close")}</button
        >
    {/snippet}</Modal
>
