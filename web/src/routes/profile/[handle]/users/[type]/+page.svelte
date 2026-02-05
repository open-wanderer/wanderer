<script lang="ts">
    import { page } from "$app/state";
    import { follows_index } from "$lib/stores/follow_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import { APIError } from "$lib/util/api_util.js";
    import { _ } from "svelte-i18n";
    let { data } = $props();

    let follows = $state(data.follows);

    $effect(() => {
        page.params.type;
        follows = data.follows;
    });

    let loading: boolean = $state(false);

    let pagination = $derived({
        page: data.follows.page,
        totalPages: data.follows.totalPages,
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
        try {
            follows = await follows_index(
                {
                    type: page.params.type as "followers" | "following",
                    username: page.params.handle!,
                },
                pagination.page,
                10,
                fetch,
            );
        } catch (e) {
            if (e instanceof APIError) {
                show_toast({
                    icon: "close",
                    text: `${e.status}: ${e.message}`,
                    type: "error"
                })
            }
        }
    }
</script>

<svelte:window onscroll={onListScroll} />

<div style="min-height: 70vh">
    <h1 class="text-3xl font-semibold mb-4">{$_(page.params.type!)}</h1>
    <ul class="space-y-4">
        {#each follows.items as follow}
            <a
                href="/profile/@{follow.preferred_username}{follow.isLocal
                    ? ''
                    : '@' + follow.domain}"
            >
                <li
                    class="flex items-center gap-x-4 hover:bg-menu-item-background-hover p-2"
                >
                    <img
                        class="rounded-full w-10 aspect-square overflow-hidden"
                        src={follow.icon ||
                            `https://api.dicebear.com/7.x/initials/svg?seed=${follow.preferred_username}&backgroundType=gradientLinear`}
                        alt="avatar"
                    />
                    <div>
                        <p class="text-lg font-medium">
                            {follow.username}
                        </p>
                        <p class="text-sm text-gray-500 break-all">
                            @{follow.preferred_username}@{follow.domain}
                        </p>
                    </div>
                </li>
            </a>
        {/each}
    </ul>
    {#if loading}
        <div class="flex items-center justify-center mt-6">
            <div class="aspect-square spinner"></div>
        </div>
    {/if}
</div>
