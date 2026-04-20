<script lang="ts">
    import { page } from "$app/state";
    import Toggle from "$lib/components/base/toggle.svelte";
    import { NotificationType } from "$lib/models/notification";
    import type { Settings } from "$lib/models/settings";
    import { settings_update } from "$lib/stores/settings_store";
    import { _ } from "svelte-i18n";

    const s =  (page.data.settings as Settings)?.notifications

    const notifications = $state({
        list_share: {
            web:  s?.list_share?.web ?? true,
            email: s?.list_share?.email ?? true,
        },
        trail_share: {
            web: s?.trail_share?.web ?? true,
            email: s?.trail_share?.email ?? true,
        },
        trail_like: {
            web: s?.trail_like?.web ?? true,
            email: s?.trail_like?.email ?? true,
        },
        new_follower: {
            web: s?.new_follower?.web ?? true,
            email: s?.new_follower?.email ?? true,
        },
        trail_comment: {
            web: s?.trail_comment?.web ?? true,
            email: s?.trail_comment?.email ?? true,
        },
        summit_log_create: {
            web: s?.summit_log_create?.web ?? true,
            email: s?.summit_log_create?.email ?? true,
        },
        trail_mention: {
            web: s?.trail_mention?.web ?? true,
            email: s?.trail_mention?.email ?? true,
        },
        comment_mention: {
            web: s?.comment_mention?.web ?? true,
            email: s?.comment_mention?.email ?? true,
        },
        summit_log_mention: {
            web: s?.summit_log_mention?.web ?? true,
            email: s?.summit_log_mention?.email ?? true,
        },
    });

    const notificationItems: { text: string; key: NotificationType }[] = [
        {
            text: $_("settings-notification-trail-comment"),
            key: NotificationType.trailComment,
        },
        {
            text: $_("settings-notification-new-follower"),
            key: NotificationType.newFollower,
        },
        {
            text: $_("settings-notification-trail-share"),
            key: NotificationType.trailShare,
        },
        {
            text: $_("settings-notification-trail-like"),
            key: NotificationType.trailLike,
        },
        {
            text: $_("settings-notification-list-share"),
            key: NotificationType.listShare,
        },
        {
            text: $_("settings-notification-summit-log-create"),
            key: NotificationType.summitLogCreate,
        },
        {
            text: $_("settings-notification-trail-mention"),
            key: NotificationType.trailMention,
        },
        {
            text: $_("settings-notification-comment-mention"),
            key: NotificationType.commentMention,
        },
        {
            text: $_("settings-notification-summit-log-mention"),
            key: NotificationType.summitLogMention,
        },
    ];

    async function updateNotificationSettings() {
        await settings_update({
            id: page.data.settings!.id,
            notifications,
        });
    }
</script>

<svelte:head>
    <title>{$_("settings")} | {page.data.appTitle}</title>
</svelte:head>
<h2 class="text-2xl font-semibold">{$_("notifications")}</h2>
<hr class="mt-4 mb-6 border-input-border" />

<div
    class="grid gap-4"
    style="grid-template-columns: 1fr min-content min-content;"
>
    <div></div>
    <span class="text-sm font-medium">Web</span>
    <span class="text-sm font-medium">Email</span>
    {#each notificationItems as item}
        <p>{item.text}</p>
        <div>
            <Toggle
                onchange={updateNotificationSettings}
                bind:value={notifications[item.key].web}
            ></Toggle>
        </div>
        <div>
            <Toggle
                onchange={updateNotificationSettings}
                bind:value={notifications[item.key].email}
            ></Toggle>
        </div>
    {/each}
</div>
