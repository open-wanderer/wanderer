<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import type { List } from "$lib/models/list";
    import { ListShare } from "$lib/models/list_share";
    import { TrailShare } from "$lib/models/trail_share";
    import {
        list_share_create,
        list_share_delete,
        list_share_index,
        list_share_update,
        shares,
    } from "$lib/stores/list_share_store";
    import {
        trail_share_create,
        trail_share_index,
    } from "$lib/stores/trail_share_store";
    import { currentUser } from "$lib/stores/user_store";
    import { _ } from "svelte-i18n";
    import ActorSearch from "../actor_search.svelte";
    import Button from "../base/button.svelte";
    import type { SelectItem } from "../base/select.svelte";
    import Select from "../base/select.svelte";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";

    interface Props {
        list: List;
        onsave?: () => void;
        onupdate?: (list: List) => void;
    }

    let { list, onsave, onupdate }: Props = $props();

    let modal: Modal;

    let displayShareError = $state(false);

    export function openModal() {
        openShareModal();
    }

    async function openShareModal() {
        await fetchShares();
        modal.openModal();
    }

    let copyButtonText = $state($_("copy-link"));

    let sharesLoading: boolean = false;

    const permissionSelectItems: SelectItem[] = [
        { text: $_("view"), value: "view" },
        { text: $_("edit"), value: "edit" },
    ];

    function copyURLToClipboard() {
        navigator.clipboard.writeText(
            `${window.location.origin}/lists/${handleFromRecordWithIRI(list)}/${list.id}`,
        );

        copyButtonText = $_("link-copied");
        setTimeout(() => (copyButtonText = $_("copy-link")), 3000);
    }

    function close() {
        onsave?.();
        modal.closeModal!();
    }

    async function shareTrails(actorIRI: string) {
        const existingTrailShares = await trail_share_index({
            actorIRI: actorIRI,
        });
        const trailIds = existingTrailShares.map((s) => s.trail);
        for (const trail of list.expand?.trails ?? []) {
            if (
                !trailIds.includes(trail.id!) &&
                !trail.public &&
                trail.author == $currentUser?.actor
            ) {
                const share = new TrailShare(actorIRI, trail.id!, "view");
                await trail_share_create(share);
            }
        }
    }

    async function shareList(item: SelectItem) {
        if (!item.value.is_local && !list.public) {
            displayShareError = true;
            return;
        }
        displayShareError = false;
        const share = new ListShare(item.value.iri, list.id!, "view");
        await list_share_create(share);
        await shareTrails(item.value.iri);
        fetchShares();
    }

    async function updateSharePermission(
        share: ListShare,
        permission: "view" | "edit",
    ) {
        share.permission = permission;
        await list_share_update(share);
    }

    async function deleteShare(share: ListShare) {
        await list_share_delete(share);
        // await deleteTrailShares(share.user);
        fetchShares();
    }

    async function fetchShares() {
        sharesLoading = true;
        const fetchedShares = await list_share_index(list.id!);
        list.expand = {
            ...list.expand,
            trails: list.expand?.trails ?? [],
            list_share_via_list: fetchedShares.items,
        };
        sharesLoading = false;
        onupdate?.(list);
    }
</script>

<Modal
    id="share-modal"
    title={$_("share-this-list")}
    size="md:min-w-sm overflow-visible"
    bind:this={modal}
>
    {#snippet content()}
        <div>
            <p
                class="p-4 bg-amber-100 rounded-xl mb-4 text-sm text-gray-500"
            >
                {$_("list-share-warning")}
            </p>
            {#if displayShareError}
                <p class="p-4 bg-red-100 rounded-xl mb-4 text-sm text-gray-500">
                    <i class="fa fa-warning mr-2"></i>
                    {$_("object-share-error", {
                        values: { object: $_("trail", { values: { n: 1 } }) },
                    })}
                </p>
            {/if}
            <ActorSearch includeSelf={false} onclick={(item) => shareList(item)}
            ></ActorSearch>
            <h4 class="font-semibold mt-4">{$_("shared-with")}</h4>

            {#if $shares.length == 0}
                <p class="text-gray-500 text-center mt-2 text-sm">
                    {$_("list-not-shared")}
                </p>
            {:else}
                {#each $shares as share}
                    {#if share.expand}
                        <div class="flex items-center gap-x-2 p-2">
                            <img
                                class="rounded-full w-8 aspect-square mr-2"
                                src={share.expand.actor.icon ||
                                    `https://api.dicebear.com/7.x/initials/svg?seed=${share.expand.actor.preferred_username}&backgroundType=gradientLinear`}
                                alt="avatar"
                            />
                            <p>
                                {`@${share.expand.actor.preferred_username}${share.expand.actor.is_local ? "" : "@" + share.expand.actor.domain}`}
                            </p>
                            <span
                                class="basis-full text-sm text-center text-gray-500"
                                >{$_("can")}</span
                            >
                            <div
                                class="shrink-0"
                                class:tooltip={!share.expand.actor.is_local}
                                data-title={$_("remote-users-cannot-edit")}
                            >
                                <Select
                                    bind:value={share.permission}
                                    items={permissionSelectItems}
                                    disabled={!share.expand.actor.is_local}
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
