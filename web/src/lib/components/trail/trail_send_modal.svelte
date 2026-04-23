<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";
    import hammerheadLogoWhite from "$lib/assets/svgs/logos/hammerhead_white.svg";
    import hammerheadLogoDark from "$lib/assets/svgs/logos/hammerhead_dark.svg";
    import { theme } from "$lib/stores/theme_store";

    interface Props {
        title?: string;
        onsend?: (settings: typeof sendSettings) => void;
    }

    let { title = $_("send-to"), onsend: onexport }: Props = $props();

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    const sendSettings: {
        integrationName: string;
    } = $state({
        integrationName: "hammerhead",
    });

    function sendTrail() {
        onexport?.(sendSettings);
        modal.closeModal();
    }
</script>

<Modal id="send-modal" {title} size="md:min-w-sm" bind:this={modal}>
    {#snippet content()}
        <div>
            <button class="btn-secondary" onclick={sendTrail}>
                <img class="h-20" src={$theme == "light" ? hammerheadLogoDark : hammerheadLogoWhite} alt="integration logo"/>
            </button>
        </div>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
        </div>
    {/snippet}</Modal
>
