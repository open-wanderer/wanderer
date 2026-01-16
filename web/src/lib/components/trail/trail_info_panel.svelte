<script lang="ts">
    import { goto } from "$app/navigation";
    import Tabs from "$lib/components/base/tabs.svelte";
    import TrailDropdown from "$lib/components/trail/trail_dropdown.svelte";
    import { Comment } from "$lib/models/comment";
    import type { Trail } from "$lib/models/trail";

    import {
        comments,
        comments_create,
        comments_delete,
        comments_index,
        comments_update,
    } from "$lib/stores/comment_store";
    import { currentUser } from "$lib/stores/user_store";
    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatTimeHHMM,
    } from "$lib/util/format_util";

    import { browser } from "$app/environment";
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import * as M from "maplibre-gl";
    import "photoswipe/style.css";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";
    import Button from "../base/button.svelte";
    import Chip from "../base/chip.svelte";
    import SkeletonNotificationCard from "../base/skeleton_notification_card.svelte";
    import Textarea from "../base/textarea.svelte";
    import CommentCard from "../comment/comment_card.svelte";
    import EmptyStateComment from "../empty_states/empty_state_comment.svelte";
    import EmptyStateDescription from "../empty_states/empty_state_description.svelte";
    import EmptyStatePhotos from "../empty_states/empty_state_photos.svelte";
    import PhotoGallery from "../photo_gallery.svelte";
    import ShareInfo from "../share_info.svelte";
    import SummitLogTable from "../summit_log/summit_log_table.svelte";
    import MapWithElevationMaplibre from "./map_with_elevation_maplibre.svelte";
    import TrailTimeline from "./trail_timeline.svelte";
    import {
        summit_logs_create,
        summit_logs_delete,
        summit_logs_index,
        summit_logs_update,
        summitLog,
        summitLogs,
    } from "$lib/stores/summit_log_store";
    import { SummitLog } from "$lib/models/summit_log";
    import SummitLogModal from "../summit_log/summit_log_modal.svelte";
    import SkeletonTable from "../base/skeleton_table.svelte";
    import ConfirmModal from "../confirm_modal.svelte";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import LikeButton from "./like_button.svelte";
    import Editor from "../base/editor.svelte";

    interface Props {
        initTrail: Trail;
        handle: string;
        mode?: "overview" | "map" | "list";
        markers?: M.Marker[];
        activeTab?: number;
    }

    let {
        initTrail,
        handle,
        mode = "map",
        markers = [],
        activeTab = 0,
    }: Props = $props();

    let summitLogModal: SummitLogModal;
    let confirmModal: ConfirmModal;

    let trail = $state(initTrail);

    const tabs = [
        $_("summit-book"),
        $_("photos"),
        ...($currentUser ? [$_("comment", { values: { n: 2 } })] : []),
    ];

    const trailIsShared =
        (trail.expand?.trail_share_via_trail?.length ?? 0) > 0;

    let gallery: PhotoGallery;

    let newComment: Comment = $state({
        text: "",
        author: "",
        trail: handle + "/" + (trail.id ?? ""),
    });

    let commentsLoading: boolean = $state(activeTab == 2);
    let commentCreateLoading: boolean = $state(false);
    let commentDeleteLoading: boolean = false;

    let summitLogsLoading: boolean = $state(activeTab == 0);
    let summitLogCreateLoading: boolean = $state(false);

    let fullDescription: boolean = $state(false);

    onMount(async () => {});

    function openMarkerPopup(i: number) {
        if ((markers[i] as M.Marker).getPopup().isOpen()) {
            return;
        }
        (markers[i] as M.Marker).togglePopup();
    }

    function closeMarkerPopup(i: number) {
        if (!(markers[i] as M.Marker).getPopup().isOpen()) {
            return;
        }
        (markers[i] as M.Marker).togglePopup();
    }

    async function toggleMapFullScreen() {
        goto(`/map/trail/${handle}/${trail.id!}`);
    }

    async function fetchComments() {
        commentsLoading = true;
        const trailId = trail.iri ? trail.iri : trail.id!;
        try {
            await comments_index(trailId, handle);
        } catch (e) {
            show_toast({
                type: "error",
                icon: "close",
                text: "Error loading comments.",
            });
        } finally {
            commentsLoading = false;
        }
    }

    async function createComment() {
        if (!$currentUser || !trail.id) {
            return;
        }
        commentCreateLoading = true;
        newComment.author = $currentUser.actor;
        newComment.trail = trail.id;

        try {
            const c = await comments_create(newComment);
            newComment.text = "";

            const newCommentList = [c, ...$comments];
            comments.set(newCommentList);
        } catch (e) {
            show_toast({
                icon: "close",
                type: "error",
                text: $_("error-posting-comment"),
            });
        } finally {
            commentCreateLoading = false;
        }
    }

    async function editComment(data: { comment: Comment; text: string }) {
        data.comment.text = data.text;
        await comments_update(data.comment);
    }

    async function deleteComment(comment: Comment) {
        commentDeleteLoading = true;
        await comments_delete(comment);
        const newCommentList = $comments.filter((c) => c.id !== comment.id);
        comments.set(newCommentList);
        commentDeleteLoading = false;
    }

    function getHeaderPhotos() {
        if (trail.photos.length) {
            return trail.photos.slice(0, 3).map((p) => getFileURL(trail, p));
        } else {
            return $theme === "light"
                ? [emptyStateTrailLight]
                : [emptyStateTrailDark];
        }
    }

    const headerPhotos = getHeaderPhotos();

    $effect(() => {
        if (browser && activeTab == 2) {
            fetchComments();
        }
    });

    $effect(() => {
        if (activeTab == 0) {
            fetchSummitLog();
        }
    });

    async function fetchSummitLog() {
        summitLogsLoading = true;
        const trailId = trail.iri ? trail.iri : trail.id;
        try {
            await summit_logs_index({ trail: trailId, category: [] }, handle);
        } catch (e) {
            show_toast({
                type: "error",
                icon: "close",
                text: "Error loading summit logs.",
            });
        } finally {
            summitLogsLoading = false;
        }
    }

    function beforeSummitLogModalOpen(currentSummitLog?: SummitLog) {
        currentSummitLog ??= new SummitLog(
            new Date().toISOString().split("T")[0],
        );

        summitLog.set(currentSummitLog);
        summitLogModal.openModal();
    }

    async function saveSummitLog(log: SummitLog) {
        summitLogCreateLoading = true;
        if (log.id) {
            let oldLogIndex = $summitLogs.findIndex((l) => l.id === log.id);
            if (oldLogIndex < 0) {
                return;
            }
            const updatedLog = await summit_logs_update(
                $summitLogs[oldLogIndex],
                log,
            );
            $summitLogs[oldLogIndex] = updatedLog;
        } else {
            log.trail = trail.id;
            const newLog = await summit_logs_create(log);
            summitLogs.set([...$summitLogs, newLog]);
        }
        summitLogCreateLoading = false;
    }

    function beforeConfirmModalOpen(currentSummitLog: SummitLog) {
        summitLog.set(currentSummitLog);
        confirmModal.openModal();
    }

    async function deleteSummitLog() {
        await summit_logs_delete($summitLog);
        const newSummitLogList = $summitLogs.filter(
            (l) => l.id !== $summitLog.id,
        );
        summitLogs.set(newSummitLogList);
    }
</script>

<div
    class="trail-info-panel mx-auto {mode == 'list'
        ? ''
        : 'border border-input-border rounded-3xl'} h-full"
    class:overflow-y-scroll={mode !== "overview"}
    style="max-width: min(100%, 76rem);"
>
    <div class="trail-info-panel-header">
        <section class="relative">
            {#if mode !== "list"}
                <button
                    aria-label="Back"
                    class="bg-black/40 text-white text-lg rounded-full w-10 h-10 hover:bg-black/50 transition-colors focus:ring-4 focus:ring-primary/70 top-6 left-6 absolute"
                    onclick={() => history.back()}
                >
                    <i class="fa fa-arrow-left"></i>
                </button>
            {/if}
            <div
                class="grid gap-[1px] {headerPhotos.length > 1
                    ? 'grid-cols-[8fr_5fr]'
                    : 'grid-cols-1'} h-80 rounded-t-3xl overflow-hidden cursor-pointer"
            >
                <PhotoGallery
                    photos={trail.photos.map((p) => getFileURL(trail, p))}
                    bind:this={gallery}
                ></PhotoGallery>
                {#each headerPhotos as photo, i}
                    {#if isVideoURL(photo)}
                        <!-- svelte-ignore a11y_media_has_caption -->
                        <video
                            class="object-cover h-full w-full"
                            onclick={trail.photos.length
                                ? () => gallery.openGallery(i)
                                : null}
                            autoplay
                            loop
                            src={photo}
                        ></video>
                    {:else}
                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
                        <img
                            class="object-cover h-full w-full"
                            onclick={trail.photos.length
                                ? () => gallery.openGallery(i)
                                : null}
                            class:row-span-2={i == 0 && headerPhotos.length > 2}
                            src={photo}
                            alt=""
                        />
                    {/if}
                {/each}
            </div>
        </section>
        <section class="border-b border-input-border p-8">
            <div class="flex justify-between items-center gap-x-4">
                {#if trail.expand?.tags && trail.expand.tags.length > 0}
                    <div class="flex flex-wrap gap-2">
                        {#each trail.expand.tags as tag}
                            <Chip text={tag.name} primary={false}></Chip>
                        {/each}
                    </div>
                {/if}
                {#if (trail.public || trailIsShared) && $currentUser}
                    <div
                        class="flex {trail.public && trailIsShared
                            ? 'w-16'
                            : 'w-8'} h-8 rounded-full items-center"
                    >
                        {#if trail.public && $currentUser}
                            <span
                                class:tooltip={mode != "map"}
                                class:mr-3={trail.public && trailIsShared}
                                data-title={$_("public")}
                            >
                                <i class="fa fa-globe"></i>
                            </span>
                        {/if}
                        {#if trailIsShared}
                            <ShareInfo type="trail" subject={trail}></ShareInfo>
                        {/if}
                    </div>
                {/if}
            </div>
            <div class="flex justify-between items-end w-full gap-y-4">
                <div class=" overflow-hidden">
                    <h4
                        title={trail.name}
                        class="{mode == 'map'
                            ? 'text-4xl'
                            : 'text-5xl'} font-bold line-clamp-3 mb-1 wrap-anywhere"
                        style="line-height: 1.18"
                    >
                        {trail.name}
                    </h4>
                    {#if trail.date}
                        <h5 class="text-sm text-gray-500">
                            {new Date(trail.date).toLocaleDateString(
                                undefined,
                                {
                                    month: "long",
                                    day: "2-digit",
                                    year: "numeric",
                                    timeZone: "UTC",
                                },
                            )}
                        </h5>
                    {/if}
                    {#if trail.expand?.author}
                        <p class="mt-3 mb-3">
                            {$_("by")}
                            <img
                                class="rounded-full w-8 aspect-square mx-1 inline"
                                src={trail.expand.author.icon ||
                                    `https://api.dicebear.com/7.x/initials/svg?seed=${trail.expand.author.preferred_username}&backgroundType=gradientLinear`}
                                alt="avatar"
                            />
                            <a class="underline" href="/profile/{handle}"
                                >{handleFromRecordWithIRI(trail)}</a
                            >
                        </p>
                    {/if}
                    <div class="flex flex-wrap gap-x-8 gap-y-2 mt-2 mr-8">
                        {#if trail.location}
                            <h3 class="text-lg">
                                <i class="fa fa-location-dot mr-2"></i>
                                {trail.location}
                            </h3>
                        {/if}
                        <h3 class="text-lg">
                            <i class="fa fa-gauge mr-2"></i>
                            {$_(trail.difficulty ?? "?")}
                        </h3>
                    </div>
                </div>
                <div class="flex flex-col items-center gap-y-2">
                    {#if ($currentUser && $currentUser.actor == trail.author) || trail.expand?.trail_share_via_trail?.length || trail.public}
                        <LikeButton {trail}></LikeButton>
                    {/if}
                    <TrailDropdown
                        trails={new Set<Trail>([trail])}
                        onDelete={() =>
                            history.length ? history.back() : goto("/trails")}
                        {mode}
                    ></TrailDropdown>
                </div>
            </div>
        </section>
        <section
            class="grid grid-cols-2 sm:grid-cols-5 gap-y-4 py-4 border-b border-input-border px-3"
        >
            <div class="flex flex-col items-center">
                <span class="font-medium text-center"
                    >{#if mode == "overview"}
                        {$_("distance")}
                    {:else}
                        <i class="fa fa-left-right"></i>
                    {/if}</span
                >
                <span class="">{formatDistance(trail.distance)}</span>
            </div>
            <div class="flex flex-col items-center">
                <span class="font-medium text-center"
                    >{#if mode == "overview"}
                        {$_("est-duration")}
                    {:else}
                        <i class="fa fa-clock"></i>
                    {/if}</span
                >
                <span class="">{formatTimeHHMM(trail.duration)}</span>
            </div>
            <div class="flex flex-col items-center">
                <span class="font-medium text-center"
                    >{#if mode == "overview"}
                        {$_("elevation-gain")}
                    {:else}
                        <i class="fa fa-arrow-trend-up"></i>
                    {/if}</span
                >
                <span class="">{formatElevation(trail.elevation_gain)}</span>
            </div>
            <div class="flex flex-col items-center">
                <span class="font-medium text-center"
                    >{#if mode == "overview"}
                        {$_("elevation-loss")}
                    {:else}
                        <i class="fa fa-arrow-trend-down"></i>
                    {/if}</span
                >
                <span class="">{formatElevation(trail.elevation_loss)}</span>
            </div>
            {#if trail.expand?.category}
                <div class="flex flex-col items-center">
                    <span class="font-medium text-center"
                        >{#if mode == "overview"}
                            {$_("category")}
                        {:else}
                            <i class="fa fa-route"></i>
                        {/if}</span
                    >
                    <span class="">{$_(trail.expand.category.name)}</span>
                </div>
            {/if}
        </section>
    </div>
    <section class="trail-info-panel-content px-8">
        <div
            class="grid grid-cols-1 my-4 gap-8"
            class:xl:grid-cols-[1fr_18rem]={mode == "overview"}
        >
            <div class="order-1 xl:-order-1">
                <h4 class="text-2xl font-semibold my-4">
                    {$_("description")}
                </h4>
                {#if trail.description?.length}
                    <article
                        class="text-justify whitespace-pre-line text-sm prose dark:prose-invert"
                    >
                        {@html !fullDescription
                            ? trail.description?.substring(0, 300)
                            : trail.description}
                        {#if (trail.description?.length ?? 0) > 300 && !fullDescription}
                            <button
                                onclick={(e) => {
                                    e.stopPropagation();
                                    e.preventDefault();
                                    fullDescription = true;
                                }}
                            >
                                ... <span class="underline"
                                    >{$_("read-more")}</span
                                ></button
                            >
                        {/if}
                    </article>
                {:else}
                    <EmptyStateDescription></EmptyStateDescription>
                {/if}
                <h4 class="text-2xl font-semibold mb-6 mt-12">
                    {$_("route", { values: { n: 1 } })}
                </h4>
                {#if mode === "overview"}
                    <div
                        class="relative border border-input-border rounded-xl p-2 mb-6 text-xs"
                        id="epc-container"
                    ></div>
                {/if}
                <TrailTimeline
                    {trail}
                    onmouseenter={openMarkerPopup}
                    onmouseleave={closeMarkerPopup}
                ></TrailTimeline>

                <div class="mb-6 mt-12 flex justify-between flex-wrap gap-y-4">
                    <Tabs {tabs} bind:activeTab></Tabs>
                    {#if $currentUser && activeTab == 0 && mode != "list"}
                        <Button
                            secondary
                            type="button"
                            loading={summitLogCreateLoading}
                            disabled={summitLogCreateLoading}
                            onclick={() => beforeSummitLogModalOpen()}
                            ><i class="fa fa-plus mr-2"></i>{$_(
                                "add-entry",
                            )}</Button
                        >
                    {/if}
                </div>
                {#if activeTab == 0}
                    <div
                        class="overflow-x-auto overflow-y-clip pb-3 scroll-x-only min-h-[175px]"
                    >
                        {#if summitLogsLoading}
                            <SkeletonTable></SkeletonTable>
                        {:else}
                            <SummitLogTable
                                {handle}
                                summitLogs={$summitLogs}
                                showAuthor
                                showRoute
                                showPhotos
                                showMenu
                                onedit={(log) => beforeSummitLogModalOpen(log)}
                                ondelete={(log) => beforeConfirmModalOpen(log)}
                            ></SummitLogTable>
                        {/if}
                    </div>
                {/if}
                {#if activeTab == 1}
                    {#if trail.photos.length}
                        <div
                            id="photo-gallery"
                            class="grid grid-cols-1 {mode == 'overview'
                                ? 'sm:grid-cols-2 md:grid-cols-3'
                                : ''} gap-4"
                        >
                            {#each trail.photos ?? [] as photo, i}
                                <!-- svelte-ignore a11y_click_events_have_key_events -->
                                <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
                                {#if isVideoURL(photo)}
                                    <!-- svelte-ignore a11y_media_has_caption -->
                                    <video
                                        controls={false}
                                        loop
                                        class="rounded-xl cursor-pointer hover:scale-105 transition-transform"
                                        onclick={() => gallery.openGallery(i)}
                                        onmouseenter={(e) =>
                                            (e.target as any).play()}
                                        onmouseleave={(e) =>
                                            (e.target as any).pause()}
                                        src={getFileURL(trail, photo)}
                                    ></video>
                                {:else}
                                    <img
                                        class="rounded-xl cursor-pointer hover:scale-105 transition-transform"
                                        onclick={() => gallery.openGallery(i)}
                                        src={getFileURL(trail, photo)}
                                        alt=""
                                    />
                                {/if}
                            {/each}
                        </div>
                    {:else}
                        <EmptyStatePhotos></EmptyStatePhotos>
                    {/if}
                {/if}
                {#if activeTab == 2}
                    <div>
                        {#if $currentUser}
                            <div class="flex items-center gap-4">
                                <img
                                    class="rounded-full w-10 aspect-square"
                                    src={getFileURL(
                                        $currentUser,
                                        $currentUser.avatar,
                                    ) ||
                                        `https://api.dicebear.com/7.x/initials/svg?seed=${$currentUser.username?.toLowerCase()}&backgroundType=gradientLinear`}
                                    alt="avatar"
                                />
                                <div class="basis-full">
                                    <Editor
                                        bind:value={newComment.text}
                                        extraClasses="min-h-24 max-w-full"
                                        placeholder="Add comment..."
                                    ></Editor>
                                </div>
                            </div>
                            <div class="flex justify-end mt-3">
                                <Button
                                    onclick={createComment}
                                    loading={commentCreateLoading}
                                    secondary={true}
                                    disabled={commentCreateLoading ||
                                        newComment.text.length == 0}
                                    >Comment</Button
                                >
                            </div>
                        {/if}
                        {#if commentsLoading}
                            {#each { length: 3 } as _, index}
                                <SkeletonNotificationCard
                                ></SkeletonNotificationCard>
                            {/each}
                        {:else if $comments.length == 0}
                            <div class="my-4">
                                <EmptyStateComment></EmptyStateComment>
                            </div>
                        {:else}
                            <ul class="space-y-4">
                                {#each $comments ?? [] as comment}
                                    <li>
                                        <CommentCard
                                            {comment}
                                            mode={comment.author ==
                                            $currentUser?.actor
                                                ? "edit"
                                                : "show"}
                                            ondelete={deleteComment}
                                            onedit={editComment}
                                        ></CommentCard>
                                    </li>
                                {/each}
                            </ul>
                        {/if}
                    </div>
                {/if}
            </div>

            {#if mode == "overview"}
                <div
                    class="block xl:sticky top-4 h-72 rounded-xl overflow-hidden"
                >
                    <MapWithElevationMaplibre
                        trails={[trail]}
                        activeTrail={0}
                        waypoints={trail.expand?.waypoints_via_trail}
                        showElevation={true}
                        elevationProfileContainer={"epc-container"}
                        showStyleSwitcher={false}
                        showFullscreen={true}
                        mapOptions={{ attributionControl: { compact: true } }}
                        onfullscreen={toggleMapFullScreen}
                        bind:markers
                    ></MapWithElevationMaplibre>
                </div>
            {/if}
        </div>
    </section>
</div>

<SummitLogModal bind:this={summitLogModal} onsave={(log) => saveSummitLog(log)}
></SummitLogModal>

<ConfirmModal
    id="confirm-summit-log-delete-modal"
    text={$_("delete-summit-log-confirm")}
    bind:this={confirmModal}
    onconfirm={deleteSummitLog}
></ConfirmModal>

<style>
    .trail-info-panel img {
        object-fit: cover;
    }
</style>
