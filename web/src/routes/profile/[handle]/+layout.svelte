<script lang="ts">
    import { invalidateAll } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "$lib/components/base/button.svelte";
    import type { DropdownItem } from "$lib/components/base/dropdown.svelte";
    import ProfileShareModal from "$lib/components/profile/profile_share_modal.svelte";
    import { follows_create, follows_delete } from "$lib/stores/follow_store";
    import { theme } from "$lib/stores/theme_store.js";
    import { currentUser } from "$lib/stores/user_store";
    import { _ } from "svelte-i18n";
    import errorDark from "$lib/assets/svgs/empty_states/error_dark.svg";
    import errorLight from "$lib/assets/svgs/empty_states/error_light.svg";
    import MetaTags from "$lib/components/base/meta_tags.svelte";

    let { data, children } = $props();

    let profileShareModal: ProfileShareModal | undefined = $state();

    let followLoading: boolean = $state(false);

    const profileLinks: DropdownItem[] = $derived([
        {
            text: $_("profile"),
            value: `/profile/${page.params.handle}`,
            icon: "user",
        },
        {
            text: $_("trail", { values: { n: 2 } }),
            value: `/profile/${page.params.handle}/trails`,
            icon: "route",
        },
        {
            text: $_("statistics"),
            value: `/profile/${page.params.handle}/stats`,
            icon: "chart-pie",
        },
    ]);

    let activeIndex = $derived(
        profileLinks.findIndex((l) => l.value === page.url.pathname),
    );

    async function follow() {
        if (!$currentUser) {
            return;
        }
        followLoading = true;

        if (data.follow) {
            await follows_delete(data.follow);
        } else {
            await follows_create(data.profile.id);
        }
        await invalidateAll();
        followLoading = false;
    }
</script>

<MetaTags
    title={`${$_("profile")} | ${page.data.appTitle}`}
    openGraph={{
        title: data.profile.username,
        description: data.profile.bio,
        url: `${page.url.origin}/profile/@${data.profile.preferredUsername}`,
        type: "profile",
        profile: {
            username: data.profile.preferredUsername,
        },
        images: [
            {
                url:
                    data.profile.icon ||
                    `https://api.dicebear.com/7.x/initials/svg?seed=${data.profile.preferredUsername}&backgroundType=gradientLinear`,
            },
        ],
    }}
/>

<div
    class="grid grid-cols-1 md:grid-cols-[356px_minmax(0,_1fr)] gap-6 max-w-6xl mx-4 md:mx-auto items-start"
>
    <div class="border border-input-border rounded-xl md:sticky top-8 md:ml-6">
        {#if data.profile}
            <div class="flex items-center gap-x-6 px-6 mt-6">
                <img
                    class="rounded-full w-16 aspect-square overflow-hidden shrink-0"
                    src={data.profile.icon ||
                        `https://api.dicebear.com/7.x/initials/svg?seed=${data.profile.preferredUsername}&backgroundType=gradientLinear`}
                    alt="avatar"
                />
                <div>
                    <h4 class="text-2xl font-semibold col-start-2">
                        {data.profile.username ?? "?"}
                    </h4>
                    <p class="text-sm text-gray-500 mb-4 break-all">
                        {data.profile.acct}
                    </p>
                </div>
            </div>
        {/if}
        <div class="px-6 mb-4">
            <p class="text-sm">
                <span class="text-gray-500">{$_("joined")}:</span>
                {new Date(data.profile.createdAt ?? "").toLocaleDateString(
                    undefined,
                    {
                        month: "2-digit",
                        day: "2-digit",
                        year: "numeric",
                        timeZone: "UTC",
                    },
                )}
            </p>
        </div>
        {#if data.profile.error}
            <p class="px-6 py-4 text-xs bg-red-200">
                <span class="font-semibold">{data.profile.acct}</span> could not
                be fetched from the remote server. Showing cached data.
            </p>
        {/if}
        <hr class="mb-4 border-input-border" />
        <div class="flex gap-x-6 items-center text-sm px-6">
            <a
                class:font-bold={page.url.pathname.endsWith("followers")}
                href="/profile/{data.profile.acct}/users/followers"
            >
                <p class="font-semibold">{data.profile.followers}</p>
                <p>{$_("followers")}</p>
            </a>
            <a
                class:font-bold={page.url.pathname.endsWith("following")}
                href="/profile/{data.profile.acct}/users/following"
            >
                <p class="font-semibold">{data.profile.following}</p>
                <p>{$_("following")}</p>
            </a>
            {#if !data.isOwnProfile && data.user}
                <div class="px-6 mt-2">
                    <Button
                        loading={followLoading}
                        disabled={followLoading}
                        primary={!data.follow}
                        secondary={!!data.follow}
                        icon={data.follow ? "check" : ""}
                        onclick={() => follow()}
                    >
                        {#if data.follow}
                            {data.follow.status == "accepted"
                                ? $_("following")
                                : $_("follow-request-pending")}
                        {:else}
                            {$_("follow")}
                        {/if}
                    </Button>
                </div>
            {/if}
        </div>

        <hr class="mt-4 border-input-border" />

        <div class="mt-6 mb-4">
            {#each profileLinks as link, i}
                <a
                    class="block mx-4 px-4 py-3 my-1 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors rounded-md"
                    class:bg-menu-item-background-hover={i == activeIndex}
                    href={link.value}
                    ><i class="fa fa-{link.icon} mr-2"></i>{link.text}</a
                >
            {/each}
        </div>
        {#if data.isOwnProfile}
            <div class="px-4 mb-4 flex flex-col gap-2">
                <button
                    class="btn-secondary basis-full"
                    onclick={() => profileShareModal?.openModal()}
                    >{$_("share-profile")}</button
                >
                <a
                    class="btn-secondary text-center basis-full"
                    href="/settings/profile">{$_("settings")}</a
                >
            </div>
            <ProfileShareModal bind:this={profileShareModal}
            ></ProfileShareModal>
        {/if}
    </div>
    <div class="md:mr-6">
        {@render children?.()}
    </div>
</div>
