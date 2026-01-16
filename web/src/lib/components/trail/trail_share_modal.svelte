<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import type { Trail } from "$lib/models/trail";
    import { TrailShare } from "$lib/models/trail_share";
    import {
        shares,
        trail_share_create,
        trail_share_delete,
        trail_share_index,
        trail_share_update,
    } from "$lib/stores/trail_share_store";
    import { _ } from "svelte-i18n";
    import ActorSearch from "../actor_search.svelte";
    import Button from "../base/button.svelte";
    import type { SelectItem } from "../base/select.svelte";
    import Select from "../base/select.svelte";
    import type { RadioItem } from "../base/radio_group.svelte";
    import RadioGroup from "../base/radio_group.svelte";
    import { TrailLinkShare } from "$lib/models/trail_link_share";
    import {
        linkShares,
        trail_link_share_create,
        trail_link_share_delete,
        trail_link_share_index,
    } from "$lib/stores/trail_link_share_store";

    interface Props {
        trail?: Trail;
        onsave?: () => void;
    }

    let { trail, onsave }: Props = $props();

    let modal: Modal;

    let displayShareError = $state(false);

    export function openModal() {
        openShareModalLocal();
    }
    async function openShareModalLocal() {
        await fetchShares();
        modal.openModal();
    }

    let copyButtonText = $state($_("copy-link"));

    let sharesLoading: boolean = false;

    const permissionSelectItems: SelectItem[] = [
        { text: $_("view"), value: "view" },
        { text: $_("edit"), value: "edit" },
    ];

    const publicShareItems: RadioItem[] = [
        {
            text: $_("limited"),
            value: "limited",
            description: $_("public-share-limited"),
        },
        {
            text: $_("everyone-with-the-link"),
            value: "everyone",
            description: $_("public-share-everyone"),
        },
    ];

    function copyURLToClipboard() {
        let link = window.location.href;
        if ($linkShares.length > 0) {
            link = link + "?share=" + $linkShares[0].token;
        }
        navigator.clipboard.writeText(link);

        copyButtonText = $_("link-copied");
        setTimeout(() => (copyButtonText = $_("copy-link")), 3000);
    }

    function close() {
        onsave?.();
        modal.closeModal!();
    }

    async function shareTrail(item: SelectItem) {
        if (trail === undefined) {
            return;
        }

        if (!item.value.isLocal && !trail.public) {
            displayShareError = true;
            return;
        }
        displayShareError = false;
        const share = new TrailShare(item.value.iri, trail.id!, "view");
        await trail_share_create(share);
        fetchShares();
    }

    async function updateSharePermission(
        share: TrailShare,
        permission: "view" | "edit",
    ) {
        share.permission = permission;
        await trail_share_update(share);
    }

    async function deleteShare(share: TrailShare) {
        await trail_share_delete(share);
        fetchShares();
    }

    async function fetchShares() {
        if (trail === undefined) return;

        sharesLoading = true;
        await trail_share_index({ trail: trail.id! });
        await trail_link_share_index(trail.id);
        sharesLoading = false;
    }

    async function handlePublicAccessChange(e: RadioItem) {
        if (trail === undefined) {
            return;
        }
        if (e.value === "everyone") {
            const share = new TrailLinkShare(trail.id!, "view");
            await trail_link_share_create(share);
        } else if (e.value === "limited" && $linkShares.length > 0) {
            const share = $linkShares[0];
            await trail_link_share_delete(share);
        }

        await trail_link_share_index(trail.id);
    }
</script>

<Modal
    id="share-modal"
    title={$_("share-this-trail")}
    size="md:min-w-sm overflow-visible"
    bind:this={modal}
>
    {#snippet content()}
        <div>
            {#if displayShareError}
                <p class="p-4 bg-red-100 rounded-xl mb-4 text-sm text-gray-500">
                    <i class="fa fa-warning mr-2"></i>
                    {$_("object-share-error", {
                        values: { object: $_("trail", { values: { n: 1 } }) },
                    })}
                </p>
            {/if}
            <ActorSearch
                includeSelf={false}
                onclick={(item) => shareTrail(item)}
            ></ActorSearch>
            <h4 class="font-semibold mt-4">{$_("shared-with")}</h4>

            {#if $shares.length == 0}
                <p class="text-gray-500 text-center mt-2 text-sm">
                    {$_("trail-not-shared")}
                </p>
            {:else}
                {#each $shares as share}
                    {#if share.expand}
                        <div class="flex items-center gap-x-2 p-2">
                            <img
                                class="rounded-full w-8 aspect-square mr-2"
                                src={share.expand.actor.icon ||
                                    `https://api.dicebear.com/7.x/initials/svg?seed=${share.expand.actor.username}&backgroundType=gradientLinear`}
                                alt="avatar"
                            />
                            <p>
                                {`@${share.expand.actor.username}${share.expand.actor.isLocal ? "" : "@" + share.expand.actor.domain}`}
                            </p>
                            <span
                                class="basis-full text-sm text-gray-500 text-end"
                                >{$_("can")}</span
                            >
                            <div
                                class="shrink-0"
                                class:tooltip={!share.expand.actor.isLocal}
                                data-title={$_("remote-users-cannot-edit")}
                            >
                                <Select
                                    value={share.permission}
                                    items={permissionSelectItems}
                                    disabled={!share.expand.actor.isLocal}
                                    onchange={(value) =>
                                        updateSharePermission(share, value)}
                                ></Select>
                            </div>

                            <button
                                aria-label="Delete share"
                                class="btn-icon text-red-500"
                                onclick={() => deleteShare(share)}
                                ><i class="fa fa-trash"></i></button
                            >
                        </div>
                    {/if}
                {/each}
            {/if}

            <h4 class="font-semibold mt-4 mb-2">{$_("public-access")}</h4>
            <RadioGroup
                name="account"
                items={publicShareItems}
                selected={$linkShares.length == 0 ? 0 : 1}
                onchange={handlePublicAccessChange}
            ></RadioGroup>
        </div>
    {/snippet}
    {#snippet footer()}
        <div class="flex justify-between items-center gap-4">
            <Button
                secondary={true}
                disabled={copyButtonText == $_("link-copied")}
                onclick={copyURLToClipboard}
            >
                <i class="fa fa-link mr-2"></i>
                {copyButtonText}
            </Button>
            <button class="btn-primary" onclick={close}>{$_("close")}</button>
        </div>
    {/snippet}</Modal
>
