<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import type { Notification } from "#lib/models/notification.ts";
    import {
        notifications_index,
        notifications_mark_as_seen,
    } from "#lib/stores/notification_store.ts";
    import { currentUser } from "#lib/stores/user_store.ts";
    import { onMount } from "svelte";
    import { fly } from "svelte/transition";
    import SkeletonNotificationCard from "../base/skeleton_notification_card.svelte";
    import NotificationCard from "./notification_card.svelte";
    import { _ } from "svelte-i18n";
    import emptyStateNotificationDark from "#lib/assets/svgs/empty_states/empty_state_notification_dark.svg";
    import emptyStateNotificationLight from "#lib/assets/svgs/empty_states/empty_state_notification_light.svg";
    import { theme } from "#lib/stores/theme_store.ts";

    let notifications: Notification[] = $state([]);

    const pagination = {
        page: page.data.notifications.page,
        totalPages: page.data.notifications.totalPages,
    };

    let loadingNextPage: boolean = $state(false);
    let isOpen = $state(false);
    let dropdownToggleElement: HTMLDivElement;

    let unreadCount = $derived(
        notifications.reduce((value, n) => (value += n.seen ? 0 : 1), 0),
    );

    $effect(() => {
        notifications = page.data.notifications.items
    })

    async function toggleMenu(e: MouseEvent) {
        e.stopPropagation();
        e.preventDefault();
        isOpen = !isOpen;
        pagination.page = 0;
        await loadNextPage();
    }

    function handleWindowClick(e: MouseEvent) {
        if (
            (e.target as HTMLElement).parentElement?.classList.contains(
                "dropdown-toggle",
            )
        ) {
            return;
        }

        isOpen = false;
    }

    async function onListScroll(e: Event) {
        const container = e.target as HTMLDivElement;
        const scrollTop = container.scrollTop;
        const scrollHeight = container.scrollHeight;
        const clientHeight = container.clientHeight;

        if (
            scrollTop + clientHeight >= scrollHeight * 0.8 &&
            pagination.page !== pagination.totalPages &&
            !loadingNextPage
        ) {            
            await loadNextPage();
        }
    }

    async function loadNextPage() {
        loadingNextPage = true;

        if (!$currentUser) {
            return;
        }
        pagination.page += 1;
        const result = await notifications_index(
            { recipient: $currentUser.actor },
            pagination.page,
        );

        notifications = result.items;
        loadingNextPage = false;
    }

    async function handleNotificationClick(data: {
        notification: Notification;
        link: string | null;
    }) {
        await notifications_mark_as_seen(data.notification);
        data.notification.seen = true;
        notifications = notifications;

        if (data.link) {
            goto(data.link);
        }
    }
    function handleEscape(event: KeyboardEvent) {
        if (!isOpen || event.key !== "Escape" || event.isComposing || event.defaultPrevented) return;
        const focused = document.activeElement;
        if (focused !== document.body && !dropdownToggleElement.parentElement?.contains(focused)) return;
        event.preventDefault();
        event.stopPropagation();
        isOpen = false;
        const trigger = dropdownToggleElement.querySelector<HTMLElement>("button, a[href]");
        if (trigger && !trigger.closest("[inert]") && trigger.getClientRects().length) trigger.focus({ preventScroll: true });
    }
</script>

<svelte:window onmouseup={handleWindowClick} onkeydown={handleEscape} />

<div class="dropdown relative">
    {#if unreadCount > 0}
        <div
            class="absolute pointer-events-none -top-[2px] left-5 text-xs rounded-full bg-content text-content-inverse px-1 text-center"
        >
            {unreadCount}{unreadCount >= 10 ? "+" : ""}
        </div>
    {/if}
    <div class="dropdown-toggle" bind:this={dropdownToggleElement}>
        <button
            aria-label="Toggle notification dropdown"
            onclick={toggleMenu}
            class="btn-icon"
        >
            <i class="fa fa-bell"></i>
        </button>
    </div>

    {#if isOpen}
        <ul
            data-fullscreen-overlay class="menu absolute bg-menu-background border border-input-border rounded-l-xl rounded-b-xl shadow-md right-0 overflow-scroll mt-4 max-h-96 w-72"
            class:none={isOpen}
            onscroll={onListScroll}
            style="z-index: 1001"
            in:fly={{ y: -10, duration: 150 }}
            out:fly={{ y: -10, duration: 150 }}
        >
            {#if loadingNextPage}
                {#each { length: 5 } as _, index}
                    <SkeletonNotificationCard></SkeletonNotificationCard>
                {/each}
            {:else if notifications.length}
                {#each notifications as notification}
                    <NotificationCard
                        onclick={(data) => handleNotificationClick(data)}
                        {notification}
                    ></NotificationCard>
                {/each}
            {:else}
                <div class="text-center p-8">
                    <img
                        class="mx-auto"
                        src={$theme === "light"
                            ? emptyStateNotificationLight
                            : emptyStateNotificationDark}
                        alt="notification empty state"
                    />
                    <p class="text-gray-500 text-sm text-center mt-2">
                        {$_("no-notifications")}
                    </p>
                </div>
            {/if}
        </ul>
    {/if}
</div>

<style>
</style>
