<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        pluginId: string;
        title: string;
        information: string;
        version?: string;
        homepageUrl?: string;
        donationUrl?: string;
        img?: string;
        status: "available" | "disabled" | "error";
        error?: string;
    }

    let {
        pluginId,
        title,
        information,
        version = "",
        homepageUrl,
        donationUrl,
        img,
        status,
        error = "",
    }: Props = $props();
    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }
</script>

<Modal
    id={`plugin-info-${pluginId}`}
    title={$_("plugin-info-title", { values: { plugin: title } })}
    bind:this={modal}
>
    {#snippet content()}
        <div class="max-h-[60dvh] space-y-5 overflow-y-auto pr-1">
            {#if img}
                <img
                    class="h-16 max-w-40 object-contain object-left"
                    src={img}
                    alt=""
                />
            {/if}
            {#if information}
                <p class="whitespace-pre-line text-sm leading-relaxed">{information}</p>
            {:else}
                <p class="text-sm text-gray-500">{$_("plugin-info-unavailable")}</p>
            {/if}
            <dl class="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-2 border-t border-separator pt-4 text-sm">
                <dt class="font-semibold">{$_("plugin-id")}</dt>
                <dd class="min-w-0 break-all">{pluginId}</dd>
                <dt class="font-semibold">{$_("plugin-version")}</dt>
                <dd>{version || "—"}</dd>
                <dt class="font-semibold">{$_("plugin-availability")}</dt>
                <dd>{$_(`plugin-status-${status}`)}</dd>
            </dl>
            {#if error}
                <p class="break-words rounded-lg border border-red-400/50 bg-red-400/10 p-3 text-sm text-red-500">
                    {error}
                </p>
            {/if}
        </div>
    {/snippet}
    {#snippet footer({ closeModal })}
        <div class="flex flex-wrap items-center justify-end gap-4">
            {#if homepageUrl || donationUrl}
                <div class="mr-auto flex flex-wrap items-center gap-3">
                    {#if homepageUrl}
                        <a
                            class="btn-icon inline-flex items-center justify-center"
                            href={homepageUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            title={$_("plugin-homepage-link")}
                            aria-label={$_("plugin-homepage-link")}
                        >
                            <i class="fa fa-globe" aria-hidden="true"></i>
                        </a>
                    {/if}
                    {#if donationUrl}
                        <a
                            class="btn-secondary inline-flex items-center gap-2"
                            href={donationUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            <i class="fa fa-hand-holding-heart" aria-hidden="true"></i>
                            {$_("plugin-donation-link")}
                        </a>
                    {/if}
                </div>
            {/if}
            <button class="btn-primary" type="button" onclick={closeModal}>
                {$_("close")}
            </button>
        </div>
    {/snippet}
</Modal>
