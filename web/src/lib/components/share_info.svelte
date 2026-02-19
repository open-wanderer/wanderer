<script lang="ts">
    import type { List } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import { currentUser } from "$lib/stores/user_store";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import { _ } from "svelte-i18n";
    import { fly } from "svelte/transition";

    interface Props {
        subject: Trail | List;
        large?: boolean;
        type: "trail" | "list";
    }

    let { subject, large = false, type }: Props = $props();

    let shareData = $derived.by(() =>
        type == "trail"
            ? (subject as Trail).expand?.trail_share_via_trail
            : (subject as List).expand?.list_share_via_list,
    );

    let showInfo: boolean = $state(false);
    let loading: boolean = $state(false);
    let subjectIsOwned: boolean = $derived(subject.author == $currentUser?.actor);
</script>

<div class="relative trail-share-info">
    <i
        class="fa fa-share-nodes"
        class:text-xl={large}
        role="button"
        tabindex="0"
        onmouseenter={() => (showInfo = true)}
        onmouseleave={() => (showInfo = false)}
    ></i>
    {#if showInfo}
        <ul
            class="menu absolute z-10 top-8 bg-menu-background border border-input-border rounded-xl shadow-md overflow-hidden p-3 space-y-3 text-content"
            in:fly={{ y: -10, duration: 150 }}
            out:fly={{ y: -10, duration: 150 }}
        >
            {#if loading}
                <div class="spinner spinner-dark"></div>
            {:else if shareData}
                {#if subjectIsOwned}
                    {#each shareData as share}
                        {#if share.expand}
                            <li>
                                <div class="flex items-center mr-8">
                                    <img
                                        class="rounded-full w-8 aspect-square mr-2"
                                        src={share.expand.actor.icon ||
                                            `https://api.dicebear.com/7.x/initials/svg?seed=${share.expand.actor.preferred_username}&backgroundType=gradientLinear`}
                                        alt="avatar"
                                    />
                                    <p class="font-semibold text-base">
                                        {`@${share.expand.actor.username}${share.expand.actor.isLocal ? "" : "@" + share.expand.actor.domain}`}
                                    </p>
                                    <span class="mx-2 text-sm text-gray-500"
                                        >{$_("can")}</span
                                    >
                                    <p class="whitespace-nowrap text-base">
                                        {#if share.permission == "view"}
                                            <i class="fa fa-eye"></i>
                                        {:else}
                                            <i class="fa fa-pen"></i>
                                        {/if}
                                    </p>
                                </div>
                            </li>
                        {/if}
                    {/each}
                {:else if subject.expand?.author}
                    <li>
                        <p class="text-xs text-gray-500 mb-2">
                            {$_("shared-by")}
                        </p>
                        <div class="flex items-center mr-8">
                            <img
                                class="rounded-full w-8 aspect-square mr-2"
                                src={subject.expand.author.icon ||
                                    `https://api.dicebear.com/7.x/initials/svg?seed=${subject.expand.author.preferred_username}&backgroundType=gradientLinear`}
                                alt="avatar"
                            />
                            <p class="font-semibold">
                                {handleFromRecordWithIRI(subject)}
                            </p>
                        </div>
                        <p class="text-xs text-gray-500 mt-2">
                            {$_("you-can")}
                        </p>
                        <p class="whitespace-nowrap">
                            {#if shareData[0].permission == "view"}
                                <i class="fa fa-eye mr-1"></i>
                                {$_("view")}
                            {:else}
                                <i class="fa fa-pen mr-1"></i>
                                {$_("edit")}
                            {/if}
                        </p>
                    </li>
                {/if}
            {/if}
        </ul>
    {/if}
</div>
