<script lang="ts">
    import {
        NotificationType,
        type Notification,
    } from "$lib/models/notification";
    import { formatTimeSince } from "$lib/util/format_util";
    import { untrack } from "svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        notification: Notification;
        onclick?: (data: {
            notification: Notification;
            link: string | null;
        }) => void;
    }

    let { notification, onclick }: Props = $props();

    const avatarSrc = untrack(() =>
        notification.expand?.author.icon
            ? notification.expand.author.icon
            : `https://api.dicebear.com/7.x/initials/svg?seed=${notification.expand?.author.preferred_username ?? ""}&backgroundType=gradientLinear`,
    );

    const timeSince = untrack(() =>
        formatTimeSince(new Date(notification.created ?? "")),
    );

    function getTitle(n: Notification) {
        switch (n.type) {
            case NotificationType.listShare:
                return $_("notification-list-share", {
                    values: { user: n.expand.author.username },
                });
            case NotificationType.newFollower:
                return $_("notification-new-follower");
            case NotificationType.trailComment:
                return $_("notification-trail-comment", {
                    values: {
                        user: n.expand.author.username,
                        trail: n.metadata?.trail_name,
                    },
                });
            case NotificationType.trailShare:
                return $_("notification-trail-share", {
                    values: { user: n.expand.author.username },
                });
            case NotificationType.summitLogCreate:
                return $_("notification-summit-log-create", {
                    values: {
                        user: n.expand.author.username,
                        trail: n.metadata?.trail_name,
                    },
                });
            case NotificationType.trailLike:
                return $_("notification-trail-like", {
                    values: {
                        user: n.expand.author.username,
                        trail: n.metadata?.trail_name,
                    },
                });
            case NotificationType.commentMention:
                return $_("notification-comment-mention", {
                    values: {
                        user: n.expand.author.username,
                        trail: n.metadata?.trail_name,
                    },
                });
            case NotificationType.trailMention:
                return $_("notification-trail-mention", {
                    values: {
                        user: n.expand.author.username,
                        trail: n.metadata?.trail_name,
                    },
                });
            case NotificationType.summitLogMention:
                return $_("notification-summit-log-mention", {
                    values: {
                        user: n.expand.author.username,
                        trail: n.metadata?.trail_name,
                    },
                });
        }
    }

    function getDescription(n: Notification) {
        switch (n.type) {
            case NotificationType.listShare:
                return n.metadata?.list ?? "";
            case NotificationType.newFollower:
                return n.expand.author.username ?? "";
            case NotificationType.trailShare:
                return n.metadata?.trail ?? "";
            default:
                return "";
        }
    }

    function getLink(n: Notification) {
        switch (n.type) {
            case NotificationType.listShare:
                return `/lists/${n.metadata?.author}/${n.metadata?.id}`;
            case NotificationType.newFollower:
                return `/profile/${n.metadata?.follower}`;
            case NotificationType.trailComment:
                return `/trail/view/${n.metadata?.trail_author}/${n.metadata?.trail_id}?t=2`;
            case NotificationType.trailShare:
                return `/trail/view/${n.metadata?.author}/${n.metadata?.id}`;
            case NotificationType.summitLogCreate:
                return `/trail/view/${n.metadata?.trail_author}/${n.metadata?.trail_id}`;
            case NotificationType.trailLike:
                return `/trail/view/${n.metadata?.trail_author}/${n.metadata?.trail_id}`;
            case NotificationType.commentMention:
                return `/trail/view/${n.metadata?.trail_author}/${n.metadata?.trail_id}?t=2`;
            case NotificationType.trailMention:
                return `/trail/view/${n.metadata?.trail_author}/${n.metadata?.trail_id}`;
            case NotificationType.summitLogMention:
                return `/trail/view/${n.metadata?.trail_author}/${n.metadata?.trail_id}`;
        }
    }

    function handleItemClick() {
        onclick?.({ notification, link });
    }
    let title = $derived(getTitle(notification));
    let description = $derived(getDescription(notification));
    let link = $derived(getLink(notification));
</script>

<li
    class="flex items-center gap-x-3 px-3 py-2 hover:bg-menu-item-background-hover relative cursor-pointer"
    role="presentation"
    onclick={handleItemClick}
>
    <img class="rounded-full w-8 aspect-square" src={avatarSrc} alt="avatar" />
    <div>
        <p
            class="text-sm {notification.seen
                ? 'font-normal'
                : 'font-semibold'} mr-3"
        >
            {title}
        </p>
        <p class="text-sm line-clamp-1">{description}</p>
        <p class="text-xs text-gray-500">
            {$_(`n-${timeSince.unit}-ago`, {
                values: { n: timeSince.value },
            })}
        </p>
    </div>

    {#if !notification.seen}
        <div
            class="bg-content w-[6px] aspect-square rounded-full absolute top-3 right-3"
        ></div>
    {/if}
</li>
