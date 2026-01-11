<script lang="ts">
    import { type Snippet } from "svelte";

    import type { List } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import { trail } from "$lib/stores/trail_store";
    import { getFileURL } from "$lib/util/file_util";
    import { _ } from "svelte-i18n";
    import Modal from "../base/modal.svelte";
    import Search, { type SearchItem } from "../base/search.svelte";
    import { theme } from "$lib/stores/theme_store";
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";

    interface Props {
        lists?: List[];
        trails?: Set<Trail> | undefined;
        children?: Snippet<[any]>;
        onchange?: (list: List) => void;
    }

    let { lists = [], trails, children, onchange }: Props = $props();

    let modal: Modal;
    let searchItems: SearchItem[] = $state([]);
    let query = $state("");
    let listsVersion = $state(0);

    export function openModal() {
        searchItems = [];
        query = "";
        modal.openModal();
    }

    function handleSelect(list: List) {
        const containsAll = listContainsAllTrails(list);
        onchange?.(list);

        if (containsAll) {
            removeTrailsFromList(list);
        } else {
            addTrailsToList(list);
        }

        listsVersion += 1;
        updateLists(query);
    }

    function handleRemove(list: List) {
        onchange?.(list);

        removeTrailsFromList(list);

        listsVersion += 1;
        updateLists(query);
    }

    function listContainsAllTrails(list: List): boolean {
        if (trails === undefined) {
            return listContainsCurrentTrail(list) ?? false;
        } else if (list.trails !== undefined) {
            for (const lTrail of trails) {
                if (!list.trails!.includes(lTrail.id!)) return false;
            }

            return true;
        }

        return false;
    }

    function listContainsCurrentTrail(list: List) {
        return list.trails?.includes($trail.id!);
    }

    function getSelectedTrailIds(): string[] {
        if (trails !== undefined) {
            return Array.from(trails)
                .map((lTrail) => lTrail.id)
                .filter((id): id is string => Boolean(id));
        }

        return $trail?.id ? [$trail.id] : [];
    }

    function addTrailsToList(list: List) {
        const selectedTrailIds = getSelectedTrailIds();
        if (!selectedTrailIds.length) {
            return;
        }

        list.trails ??= [];
        for (const id of selectedTrailIds) {
            if (!list.trails.includes(id)) {
                list.trails = [...list.trails, id];
            }
        }
    }

    function removeTrailsFromList(list: List) {
        const selectedTrailIds = getSelectedTrailIds();
        if (!selectedTrailIds.length || !list.trails?.length) {
            return;
        }

        list.trails = list.trails.filter(
            (id) => !selectedTrailIds.includes(id),
        );
    }

    const listsContainingTrails = $derived.by(() =>
        lists.filter((list) => {
            listsVersion;
            return listContainsAllTrails(list);
        }),
    );

    const showNoListHint = $derived.by(() => {
        if (query.trim().length) {
            return false;
        }

        if (trails !== undefined) {
            return trails.size > 0 && listsContainingTrails.length === 0;
        }

        return Boolean($trail?.id) && listsContainingTrails.length === 0;
    });

    function updateLists(q: string) {
        query = q;
        const normalizedQuery = q.trim().toLowerCase();
        if (!normalizedQuery.length) {
            searchItems = [];
            return;
        }

        searchItems = lists
            .filter((list) => {
                const name = list.name?.toLowerCase() ?? "";
                return (
                    name.includes(normalizedQuery) &&
                    !listContainsAllTrails(list)
                );
            })
            .map((list) => ({
                text: list.name,
                description: list.description,
                value: list,
                icon: list.avatar
                    ? getFileURL(list, list.avatar)
                    : $theme === "light"
                      ? emptyStateTrailLight
                      : emptyStateTrailDark,
            }));
    }

    const children_render = $derived(children);
</script>

<Modal
    id="list-search-modal"
    title={$_("select-list")}
    size="md:min-w-sm"
    bind:this={modal}
>
    {#snippet children({ openModal })}
        {@render children_render?.({ openModal })}
    {/snippet}
    {#snippet content()}
        <div class="space-y-4 min-h-40">
            <Search
                onupdate={(q) => updateLists(q)}
                timeBetweenUpdates={0}
                onclick={(item) => handleSelect(item.value)}
                placeholder={$_("search-list")}
                items={searchItems}
                bind:value={query}
            >
                {#snippet prepend({ item })}
                    <img
                        class="rounded-full w-8 aspect-square mr-2"
                        src={item.icon}
                        alt="avatar"
                    />
                {/snippet}
            </Search>
            {#if query.trim().length && searchItems.length === 0}
                <p class="text-sm text-gray-500 text-center">
                    {$_("no-results")}
                </p>
            {/if}
            {#if listsContainingTrails.length}
                <div class="space-y-2">
                    <h4 class="text-sm font-semibold text-gray-500">
                        {$_("linked-lists")}
                    </h4>
                    {#each listsContainingTrails as list}
                        <div
                            class="w-full flex items-center gap-4 p-4 rounded-xl bg-menu-item-background"
                        >
                            <img
                                class="w-12 aspect-square rounded-full"
                                src={list.avatar
                                    ? getFileURL(list, list.avatar)
                                    : $theme === "light"
                                      ? emptyStateTrailLight
                                      : emptyStateTrailDark}
                                alt="avatar"
                            />

                            <h5 class="text-md font-semibold text-left">
                                {list.name}
                            </h5>

                            <button
                                class="ml-auto btn-icon text-red-500"
                                onclick={() => handleRemove(list)}
                                type="button"
                                aria-label={$_("unlink")}
                            >
                                <i class="fa fa-trash"></i>
                            </button>
                        </div>
                    {/each}
                </div>
            {:else if showNoListHint}
                <div class="min-h-20 flex items-center justify-center">
                    <p class="text-sm text-gray-500 text-center">
                        {$_("trail-not-in-list")}
                    </p>
                </div>
            {/if}
        </div>
    {/snippet}
    {#snippet footer({ closeModal })}
        <div class="flex justify-end">
            <button class="btn-primary" onclick={closeModal}>
                {$_("close")}
            </button>
        </div>
    {/snippet}
</Modal>
