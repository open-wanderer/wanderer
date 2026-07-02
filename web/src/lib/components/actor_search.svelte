<script lang="ts">
    import type { Actor, ActorSearchResult } from "$lib/models/activitypub/actor";
    import { searchActors } from "$lib/stores/search_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { _ } from "svelte-i18n";
    import Search, { type SearchItem } from "./base/search.svelte";

    interface Props {
        label?: string;
        value?: string;
        includeSelf?: boolean;
        clearAfterSelect?: boolean;
        onclear?: () => void;
        onclick?: (item: SearchItem) => void;
    }

    let {
        label = "",
        value = $bindable(""),
        includeSelf = true,
        clearAfterSelect = true,
        onclear,
        onclick,
    }: Props = $props();

    let searchItems: SearchItem[] = $state([]);

    async function updateActors(q: string) {
        if (!q.length) {
            searchItems = [];
            onclear?.();
            return;
        }
        try {
            const actors: ActorSearchResult[] = await searchActors(q, includeSelf);
            searchItems = actors.map((a) => ({
                text: a.username,
                description: `@${a.preferred_username}${a.is_local ? "" : "@" + a.domain}`,
                value: a,
                icon: "user",
            }));
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: "Error during search",
            });
        }
    }

    function onClick(item: SearchItem) {
        value = item.value.preferred_username ?? value;

        onclick?.(item);
        searchItems = [];
    }
</script>

<Search
    onupdate={(q) => updateActors(q)}
    onclick={(item) => onClick(item)}
    placeholder={`${$_("username")}...`}
    items={searchItems}
    {clearAfterSelect}
    {label}
    bind:value
>
    {#snippet prepend({ item })}
        <img
            class="rounded-full w-8 aspect-square mr-2"
            src={item.value.icon ||
                `https://api.dicebear.com/7.x/initials/svg?seed=${item.value.preferred_username}&backgroundType=gradientLinear`}
            alt="avatar"
        />
    {/snippet}
</Search>
