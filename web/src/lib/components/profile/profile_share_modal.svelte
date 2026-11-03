<script lang="ts">
    import { page } from "$app/state";
    import Modal from "#lib/components/base/modal.svelte";
    import QrCodeWithLogo from "#lib/vendor/qr-code-with-logos/index.js";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";
    import TextField from "../base/text_field.svelte";
    import { show_toast } from "#lib/stores/toast_store.svelte.ts";

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    let qrcode: any | null = null;

    onMount(() => {
        qrcode = new QrCodeWithLogo({
            content: page.url.href,
            image: document.getElementById("qrcode") as HTMLImageElement,
            logo: {
                src: "/favicon.png",
            },
        });
    });

    function copyURLToClipboard() {
        navigator.clipboard.writeText(page.url.href);
        show_toast({
            text: $_("link-copied"),
            icon: "circle-info",
            type: "info",
        });
    }

    function downloadQRCode() {
        if (!qrcode) {
            return;
        }
        qrcode.downloadImage();
    }
</script>

<Modal
    id="password-modal"
    size="md:max-w-xl"
    title={$_("share-profile")}
    bind:this={modal}
>
    {#snippet content()}
        <div class="flex">
            <div class="space-y-2">
                <img class="w-64" id="qrcode" alt="QR Code" />
                <button
                    class="btn-secondary w-full"
                    onclick={() => downloadQRCode()}
                >
                    <i class="fa fa-download mr-2"></i>
                    <span>{$_("download")}</span>
                </button>
            </div>
            <div class="space-y-4">
                <p>
                    Send this link or the QR code on the left to someone to
                    share your profile.
                </p>
                <div class="flex gap-x-2 items-center">
                    <div class="basis-full">
                        <TextField value={page.url.href} disabled></TextField>
                    </div>
                    <button
                    aria-label="Copy profile URL"
                    class="btn-icon" onclick={copyURLToClipboard}>
                        <i class="fa-regular fa-copy"></i>
                    </button>
                </div>
            </div>
        </div>
    {/snippet}
</Modal>
