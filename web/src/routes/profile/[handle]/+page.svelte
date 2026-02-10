<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import FeedCard from "$lib/components/profile/feed_card.svelte";
    import type { FeedItem } from "$lib/models/feed.js";

    import { profile_feed_index } from "$lib/stores/profile_store.js";
    import { theme } from "$lib/stores/theme_store.js";
    import { getFileURL } from "$lib/util/file_util.js";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    $effect(() => {
        data.feed;
        feed = data.feed
    })

    let feed = $state(data.feed);

    let loading: boolean = false;

    let pagination = $derived({
        page: feed.page,
        totalPages: feed.totalItems,
    });

    async function onListScroll(e: Event) {
        if (
            window.innerHeight + window.scrollY >=
                0.8 * document.body.offsetHeight &&
            pagination.page !== pagination.totalPages &&
            !loading
        ) {
            loading = true;
            await loadNextPage();
            loading = false;
        }
    }

    async function loadNextPage() {
        pagination.page += 1;
        feed = await profile_feed_index(page.params.handle!, pagination.page);
    }

    function handleFeedItemClick(f: FeedItem) {
        const author = f.expand.item.expand?.author;
        if (f.type == "trail") {
            goto(
                `/trail/view/@${author?.preferred_username}@${author?.domain}/${f.item}`,
            );
        } else {
            goto(
                `/lists/@${author?.preferred_username}@${author?.domain}/${f.item}`,
            );
        }
    }
</script>

<svelte:window onscroll={onListScroll} />

<svelte:head>
    <title>{$_("profile")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <div class="space-y-4">
        <h4 class="text-xl font-semibold">
            {$_("about")}
            {data.profile.username}
            {#if data.isOwnProfile && data.profile.bio.length}
                <a aria-label="Edit bio" class="ml-4" href="/settings/profile"
                    ><i class="fa fa-pen text-base"></i></a
                >
            {/if}
        </h4>
        {#if data.profile.bio.length}
            <p class="prose dark:prose-invert">{@html data.profile.bio}</p>
        {:else if data.isOwnProfile}
            <a class="btn-primary inline-block" href="/settings/profile"
                >+ {$_("add-bio")}</a
            >
        {:else}
            <p class="w-full text-center text-gray-500 text-sm">
                {$_("empty-bio", {
                    values: { username: data.profile.preferredUsername },
                })}
            </p>
        {/if}
    </div>
    <div class="space-y-4">
        <h4 class="text-xl font-semibold">
            {$_("list", { values: { n: 2 } })}
        </h4>
        <div class="flex gap-x-4 overflow-x-scroll">
            {#if !data.lists.length && data.isOwnProfile}
                <a class="btn-primary inline-block" href="/lists/edit/new"
                    >+ {$_("new-list")}</a
                >
            {:else if !data.lists.length}
                <p class="w-full text-center text-gray-500 text-sm">
                    {$_("empty-lists", {
                        values: { username: data.profile.preferredUsername },
                    })}
                </p>
            {/if}
            {#each data.lists as list}
                <a
                    href="/lists/{page.params.handle}/{list.id}"
                    class="relative w-64 h-48 rounded-xl overflow-hidden group shrink-0"
                >
                    <img
                        class="w-full h-full object-cover transition-transform group-hover:scale-110"
                        src={list.avatar
                            ? getFileURL(list, list.avatar)
                            : $theme === "light"
                              ? emptyStateTrailLight
                              : emptyStateTrailDark}
                        alt="avatar"
                    />

                    <div
                        class="absolute bottom-0 w-full h-2/3 bg-gradient-to-b from-transparent to-black opacity-50"
                    ></div>
                    <h5
                        class="absolute bottom-4 left-4 font-semibold text-white"
                    >
                        {list.name}
                    </h5>
                </a>
            {/each}
        </div>
    </div>
    <div class="space-y-4">
        <h4 class="text-xl font-semibold">Timeline</h4>
        {#if !feed.items?.length && data.isOwnProfile}
            <a class="btn-primary inline-block" href="/trail/edit/new"
                >+ {$_("new-trail")}</a
            >
        {:else if !feed.items?.length}
            <p class="w-full text-center text-gray-500 text-sm">
                {$_("empty-activities", {
                    values: { username: data.profile.preferredUsername },
                })}
            </p>
        {/if}
        {#each feed.items as item}
            <div
                class="py-1 cursor-pointer"
                role="presentation"
                onclick={() => handleFeedItemClick(item)}
            >
                <FeedCard feedItem={item}></FeedCard>
            </div>
        {/each}
    </div>
</div>
