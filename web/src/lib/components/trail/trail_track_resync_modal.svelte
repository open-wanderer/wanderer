<script module lang="ts">
    export type {
        TrailTrackResyncPatch,
        TrailTrackResyncTarget,
    } from "./trail_track_resync";
</script>

<script lang="ts">
    import { plugin_track_resync, type TrackResyncKind } from "$lib/stores/plugin_instance_store";
    import { trails_update_map_cache } from "$lib/stores/trail_store";
    import { pluginErrorRetryAfterSeconds, translatePluginAPIError } from "$lib/util/plugin_error_i18n";
    import { onDestroy, tick } from "svelte";
    import { _ } from "svelte-i18n";
    import {
        applyTrackResyncPatch,
        prepareTrackResyncPatch,
        snapshotTrackResyncTarget,
        type TrailTrackResyncPatch,
        type TrailTrackResyncSnapshot,
        type TrailTrackResyncTarget as OpenTarget,
    } from "./trail_track_resync";

    interface Props {
        onupdated?: (
            trailId: string,
            patch: TrailTrackResyncPatch,
        ) => void;
    }
    type Context = { target: TrailTrackResyncSnapshot; kind: TrackResyncKind; retryDeadline: number };
    type RunState = Context &
        (
            | { phase: "confirm" | "running" }
            | { phase: "error"; message: string }
            | { phase: "done"; pageStale: boolean; warning: boolean }
        );
    let { onupdated }: Props = $props();
    const componentId = $props.id();
    const titleId = `trail-track-resync-title-${componentId}`;

    let dialog: HTMLDialogElement | undefined = $state();
    let runState: RunState | undefined = $state();
    let retryIn = $state(0);
    let previousBodyOverflow: string | undefined;
    let returnFocus: HTMLElement | undefined;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let session = 0;
    const retryDeadlines = new Map<string, number>();

    export function open(target: OpenTarget) {
        if (!dialog || dialog.open || !target.preview.available ||
            !target.trail.id || target.trail.id !== target.trailId) {
            return;
        }

        const snapshot = snapshotTrackResyncTarget(target);
        const retryDeadline = Math.max(target.retryDeadline,
            retryDeadlines.get(snapshot.key) ?? 0);
        if (retryDeadline > Date.now()) {
            retryDeadlines.set(snapshot.key, retryDeadline);
        }
        runState = {
            phase: "confirm",
            target: snapshot,
            kind: snapshot.preview.suggestedKind ?? "completed",
            retryDeadline,
        };
        session += 1;
        returnFocus =
            document.activeElement instanceof HTMLElement
                ? document.activeElement
                : undefined;
        lockPage();
        dialog.showModal();
        syncRetryTimer();
        void tick().then(() => focusButton("initial"));
    }

    function lockPage() {
        previousBodyOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";
    }

    function releasePage() {
        if (previousBodyOverflow === undefined) return;
        document.body.style.overflow = previousBodyOverflow;
        previousBodyOverflow = undefined;
    }

    function close() {
        if (dialog?.open && runState?.phase !== "running") dialog.close();
    }

    function handleCancel(event: Event) {
        event.preventDefault();
        close();
    }

    function handleClose() {
        session += 1;
        stopRetryTimer();
        releasePage();
        runState = undefined;
        const target = returnFocus;
        returnFocus = undefined;
        if (target?.isConnected) target.focus({ preventScroll: true });
    }

    function focusButton(name: "initial" | "result") {
        dialog
            ?.querySelector<HTMLButtonElement>(`[data-resync-focus="${name}"]`)
            ?.focus();
    }

    function selectKind(kind: TrackResyncKind) {
        if (runState?.phase === "confirm" || runState?.phase === "error") {
            runState = { ...runState, kind };
        }
    }

    function stopRetryTimer() {
        if (retryTimer) clearTimeout(retryTimer);
        retryTimer = undefined;
        retryIn = 0;
    }

    function syncRetryTimer() {
        stopRetryTimer();
        if (
            !dialog?.open ||
            !runState ||
            (runState.phase !== "confirm" && runState.phase !== "error")
        ) {
            return;
        }
        retryIn = Math.max(
            0,
            Math.ceil((runState.retryDeadline - Date.now()) / 1000),
        );
        if (!retryIn) {
            retryDeadlines.delete(runState.target.key);
            return;
        }
        retryTimer = setTimeout(syncRetryTimer, 1000);
    }

    async function run() {
        const current = runState;
        if (
            !current ||
            (current.phase !== "confirm" && current.phase !== "error") ||
            current.retryDeadline > Date.now()
        ) {
            return;
        }

        const currentSession = session;
        runState = {
            phase: "running",
            target: current.target,
            kind: current.kind,
            retryDeadline: current.retryDeadline,
        };
        stopRetryTimer();
        try {
            const result = await plugin_track_resync(
                current.target.trail.id!,
                current.target.preview,
                current.target.preview.kindRequired ? current.kind : undefined,
            );
            if (currentSession !== session) return;
            const applied = await prepareTrackResyncPatch(
                current.target,
                result.track,
            );
            if (currentSession !== session) return;

            runState = {
                phase: "done",
                target: current.target,
                kind: current.kind,
                retryDeadline: 0,
                pageStale: applied.pageStale,
                warning: Boolean(result.warning),
            };
            retryDeadlines.delete(current.target.key);
            trails_update_map_cache(
                applyTrackResyncPatch(current.target.trail, applied.patch),
            );
            try {
                onupdated?.(current.target.trail.id!, applied.patch);
            } catch (error) {
                console.error(error);
            }
            await tick();
            focusButton("result");
        } catch (error) {
            if (currentSession !== session) return;
            const retryDeadline = Math.max(
                current.retryDeadline,
                Date.now() + pluginErrorRetryAfterSeconds(error) * 1000,
            );
            if (retryDeadline > Date.now()) {
                retryDeadlines.set(current.target.key, retryDeadline);
            }
            runState = {
                phase: "error",
                target: current.target,
                kind: current.kind,
                retryDeadline,
                message: translatePluginAPIError(
                    error,
                    $_("plugin-track-resync-error"),
                ),
            };
            syncRetryTimer();
            await tick();
            focusButton("result");
        }
    }

    onDestroy(() => {
        session += 1;
        stopRetryTimer();
        releasePage();
    });
</script>

<dialog
    bind:this={dialog}
    aria-labelledby={titleId}
    oncancel={handleCancel}
    onclose={handleClose}
    class="max-w-2xl max-h-full rounded-xl text-content"
>
    <div class="bg-background shadow rounded-xl">
        <header
            class="flex items-center justify-between p-4 md:p-5 border-b border-separator rounded-t"
        >
            <h3 id={titleId} class="text-xl font-semibold">
                {$_("plugin-track-resync-title")}
            </h3>
            <button
                type="button"
                class="rounded-full btn-icon"
                disabled={runState?.phase === "running"}
                onclick={close}
            >
                <i class="fa fa-close" aria-hidden="true"></i>
                <span class="sr-only">{$_("close")}</span>
            </button>
        </header>

        <div class="p-4 md:p-5 space-y-4">
            <span class="sr-only" role="status" aria-live="polite" aria-atomic="true">
                {#if runState?.phase === "running"}
                    {$_("plugin-track-resync-running")}
                {:else if runState?.phase === "done"}
                    {$_("plugin-track-resync-done")}
                    {runState.pageStale ? ` ${$_("plugin-track-resync-page-stale")}` : ""}
                    {runState.warning ? ` ${$_("plugin-track-resync-warning")}` : ""}
                {:else if runState?.phase === "error"}
                    {runState.message}
                {/if}
            </span>

            {#if runState?.phase === "done"}
                <p>{$_("plugin-track-resync-done")}</p>
                {#if runState.pageStale}
                    <p class="text-sm text-amber-600 dark:text-amber-400">
                        <i class="fa fa-triangle-exclamation mr-2" aria-hidden="true"></i>
                        {$_("plugin-track-resync-page-stale")}
                    </p>
                {/if}
                {#if runState.warning}
                    <p class="text-sm text-amber-600">
                        <i class="fa fa-triangle-exclamation mr-2" aria-hidden="true"></i>
                        {$_("plugin-track-resync-warning")}
                    </p>
                {/if}
            {:else if runState}
                <p>
                    {$_("plugin-track-resync-confirm", {
                        values: {
                            provider: runState.target.preview.provider ?? "",
                            externalId: runState.target.preview.externalId ?? "",
                        },
                    })}
                </p>
                {#if runState.target.preview.originUnverified}
                    <p class="text-sm text-amber-600 dark:text-amber-400" role="note">
                        <i class="fa fa-triangle-exclamation mr-2" aria-hidden="true"></i>
                        {$_("plugin-track-resync-origin-unverified")}
                    </p>
                {/if}
                {#if runState.target.preview.kindRequired}
                    <fieldset class="space-y-2" disabled={runState.phase === "running"}>
                        <legend class="text-sm font-medium">
                            {$_("plugin-track-resync-kind-question")}
                        </legend>
                        {#each ["completed", "planned"] as kind}
                            <label class="flex items-center gap-2 text-sm">
                                <input
                                    type="radio"
                                    name={`track-resync-kind-${componentId}`}
                                    value={kind}
                                    checked={runState.kind === kind}
                                    onchange={() => selectKind(kind as TrackResyncKind)}
                                />
                                {$_(`plugin-track-resync-kind-${kind}`)}
                            </label>
                        {/each}
                    </fieldset>
                {/if}
                {#if runState.phase === "running"}
                    <p class="text-sm text-gray-500">
                        <i class="fa fa-spinner fa-spin mr-2" aria-hidden="true"></i>
                        {$_("plugin-track-resync-running")}
                    </p>
                {:else if runState.phase === "error"}
                    <p class="text-sm text-red-500">
                        <i class="fa fa-circle-exclamation mr-2" aria-hidden="true"></i>
                        {runState.message}
                    </p>
                {/if}
                {#if retryIn > 0 && runState.phase !== "running"}
                    <p class="text-sm text-gray-500">
                        {$_("plugin-track-resync-retry-in", {
                            values: { seconds: retryIn },
                        })}
                    </p>
                {/if}
            {/if}
        </div>

        <footer class="p-4 md:p-5 border-t border-separator rounded-b">
            <div class="flex items-center gap-4">
                {#if runState?.phase === "done"}
                    <button
                        type="button"
                        data-resync-focus="result"
                        class="btn-secondary"
                        onclick={close}
                    >
                        {$_("close")}
                    </button>
                    <button
                        type="button"
                        class="btn-primary"
                        onclick={() => window.location.reload()}
                    >
                        {$_("plugin-track-resync-reload")}
                    </button>
                {:else if runState}
                    <button
                        type="button"
                        data-resync-focus="initial"
                        class="btn-secondary"
                        disabled={runState.phase === "running"}
                        onclick={close}>{$_("cancel")}</button
                    >
                    <button
                        type="button"
                        data-resync-focus={runState.phase === "error" ? "result" : undefined}
                        class="btn-primary"
                        disabled={runState.phase === "running" || retryIn > 0}
                        onclick={run}
                    >
                        {runState.phase === "error"
                            ? $_("plugin-track-resync-retry")
                            : $_("plugin-track-resync-action")}
                    </button>
                {/if}
            </div>
        </footer>
    </div>
</dialog>

<style lang="postcss">
    @reference "tailwindcss";
    dialog::backdrop {
        @apply bg-gray-500 opacity-50;
    }
</style>
