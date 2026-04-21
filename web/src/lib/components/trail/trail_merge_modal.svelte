<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import Select, { type SelectItem } from "$lib/components/base/select.svelte";
    import { trails_show } from "$lib/stores/trail_store";
    import type { Trail } from "$lib/models/trail";
    import {
        trail_merge_suggest_auto,
        trail_merge_suggest_manual,
        type TrailMergeSuggestCandidate,
    } from "$lib/stores/trail_merge_api";
    import { translateTrailMergeError } from "$lib/stores/trail_merge_i18n";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { _ } from "svelte-i18n";
    import { APIError } from "$lib/util/api_util";

    export interface MergeSelection {
        targetTrail: Trail;
        sourceTrails: Trail[];
    }

    export interface OpenMergeModalOptions {
        preferredTargetTrailId?: string;
        fixedTargetTrailId?: string;
    }

    interface Props {
        title?: string;
        onmerge?: (settings: MergeSettings, selection: MergeSelection) => void;
    }

    let { title = $_("link-as-summit-log"), onmerge: onmerge }: Props = $props();

    const STORAGE_KEY_SETTINGS = "trail_merge_settings";
    const STORAGE_KEY_REMEMBER = "trail_merge_remember";

    let modal: Modal;
    let loading = $state(false);
    let candidateWarnings: Record<string, string[]> = $state({});
    let candidateReasons: Record<string, string> = $state({});
    let selectableTargets: Trail[] = $state([]);
    let targetTrailId = $state("");
    let modalTitle = $state("");
    let autoDiscoveryMode = $state(false);
    let fixedTargetSelection = $state(false);
    let mergeTrailsSelection: Trail[] = $state([]);
    let autoDiscoverySourceTrail: Trail | undefined = $state();
    let autoDiscoveryCandidateTrails: Trail[] = $state([]);

    const defaultSettings: MergeSettings = {
        summitLog: true,
        photos: true,
        comments: true,
        delete: true,
        tags: true,
        likes: true,
    };

    let rememberSettings = $state(false);

    function applySettings(next: MergeSettings) {
        settings.summitLog = next.summitLog;
        settings.photos = next.photos;
        settings.comments = next.comments;
        settings.delete = next.delete;
        settings.tags = next.tags;
        settings.likes = next.likes;
    }

    function loadStoredSettings() {
        if (typeof localStorage === "undefined") {
            applySettings(defaultSettings);
            return;
        }

        rememberSettings = localStorage.getItem(STORAGE_KEY_REMEMBER) === "true";
        if (!rememberSettings) {
            applySettings(defaultSettings);
            return;
        }

        const raw = localStorage.getItem(STORAGE_KEY_SETTINGS);
        if (!raw) {
            applySettings(defaultSettings);
            return;
        }

        try {
            const parsed = JSON.parse(raw) as Partial<MergeSettings>;
            applySettings({
                summitLog: parsed.summitLog ?? defaultSettings.summitLog,
                photos: parsed.photos ?? defaultSettings.photos,
                comments: parsed.comments ?? defaultSettings.comments,
                delete: parsed.delete ?? defaultSettings.delete,
                tags: parsed.tags ?? defaultSettings.tags,
                likes: parsed.likes ?? defaultSettings.likes,
            });
        } catch {
            applySettings(defaultSettings);
        }
    }

    function persistSettingsPreference() {
        if (typeof localStorage === "undefined") {
            return;
        }

        localStorage.setItem(STORAGE_KEY_REMEMBER, rememberSettings ? "true" : "false");
        if (!rememberSettings) {
            localStorage.removeItem(STORAGE_KEY_SETTINGS);
            return;
        }

        localStorage.setItem(
            STORAGE_KEY_SETTINGS,
            JSON.stringify({
                summitLog: settings.summitLog,
                photos: settings.photos,
                comments: settings.comments,
                delete: settings.delete,
                tags: settings.tags,
                likes: settings.likes,
            } satisfies MergeSettings),
        );
    }

    async function loadAutoDiscoveryTargets(
        sourceTrail: Trail,
    ): Promise<{
        targetTrailId: string;
        warnings: Record<string, string[]>;
        reasons: Record<string, string>;
        selectableTargets: Trail[];
    }> {
        if (!sourceTrail.id) {
            throw new Error($_("trail_merge_missing_source_trail_id"));
        }

        const response = await trail_merge_suggest_auto(sourceTrail.id);
        const selectableCandidates = response.candidates.filter(
            (candidate) => candidate.selectable,
        );

        const loadedTargets = await Promise.all(
            selectableCandidates.map(async (candidate) => {
                const trail = await trails_show(candidate.trailId);
                return { candidate, trail };
            }),
        );

        const warnings: Record<string, string[]> = {};
        const reasons: Record<string, string> = {};

        for (const { candidate } of loadedTargets) {
            warnings[candidate.trailId] = candidate.warnings;
            reasons[candidate.trailId] = candidate.reason;
        }

        const targets = loadedTargets.map(({ trail }) => trail);

        return {
            targetTrailId:
                targets.find((trail) => trail.id === response.targetTrailId)?.id ??
                targets[0]?.id ??
                "",
            warnings,
            reasons,
            selectableTargets: targets,
        };
    }

    export async function openModal(trails: Trail[] = [], options: OpenMergeModalOptions = {}) {
        loading = true;
        modalTitle = title;
        autoDiscoveryMode = false;
        fixedTargetSelection = false;
        mergeTrailsSelection = trails;
        autoDiscoverySourceTrail = undefined;
        autoDiscoveryCandidateTrails = [];
        candidateWarnings = {};
        candidateReasons = {};
        selectableTargets = [];
        targetTrailId = "";

        loadStoredSettings();
        modal.openModal();

        if (options.fixedTargetTrailId) {
            const fixedTargetTrail = trails.find((trail) => trail.id === options.fixedTargetTrailId);
            if (!fixedTargetTrail?.id) {
                show_toast({
                    type: "error",
                    icon: "close",
                    text: $_("trail-merge-no-editable-target"),
                });
                modal.closeModal();
                loading = false;
                return;
            }

            fixedTargetSelection = true;
            selectableTargets = [fixedTargetTrail];
            targetTrailId = fixedTargetTrail.id;
            loading = false;
            return;
        }

        try {
            const response = await trail_merge_suggest_manual(
                trails.map((trail) => trail.id!).filter(Boolean),
            );

            const nextWarnings: Record<string, string[]> = {};
            const nextReasons: Record<string, string> = {};

            const candidatesById = new Map<string, TrailMergeSuggestCandidate>(
                response.candidates.map((candidate) => [candidate.trailId, candidate]),
            );

            const nextSelectableTargets = trails.filter((trail) => {
                const candidate = candidatesById.get(trail.id!);
                if (!candidate) {
                    return false;
                }

                nextWarnings[trail.id!] = candidate.warnings;
                nextReasons[trail.id!] = candidate.reason;
                return candidate.selectable;
            });

            candidateWarnings = nextWarnings;
            candidateReasons = nextReasons;
            selectableTargets = nextSelectableTargets;

            if (selectableTargets.length === 0) {
                show_toast({
                    type: "error",
                    icon: "close",
                    text: $_("trail-merge-no-editable-target"),
                });
                modal.closeModal();
                return;
            }

            targetTrailId =
                selectableTargets.find((trail) => trail.id === options.preferredTargetTrailId)?.id
                ?? selectableTargets.find((trail) => trail.id === response.targetTrailId)?.id
                ?? selectableTargets[0]?.id
                ?? "";
        } catch (error) {
            console.error("Failed to suggest trail merge target", error);
            const text =
                error instanceof APIError
                    ? translateTrailMergeError(error.message)
                    : $_("trail-merge-suggest-error");
            show_toast({
                type: "error",
                icon: "close",
                text,
            });
            modal.closeModal();
            return;
        } finally {
            loading = false;
        }
    }

    export async function openSimilarTrailsModal(sourceTrail: Trail) {
        loading = true;
        modalTitle = $_("find-similar-trails");
        autoDiscoveryMode = true;
        mergeTrailsSelection = [sourceTrail];
        autoDiscoverySourceTrail = sourceTrail;
        candidateWarnings = {};
        candidateReasons = {};
        selectableTargets = [];
        autoDiscoveryCandidateTrails = [];
        targetTrailId = "";

        loadStoredSettings();
        modal.openModal();

        try {
            const result = await loadAutoDiscoveryTargets(sourceTrail);
            candidateWarnings = result.warnings;
            candidateReasons = result.reasons;
            autoDiscoveryCandidateTrails = result.selectableTargets;
            selectableTargets = [sourceTrail, ...result.selectableTargets];
            candidateWarnings[sourceTrail.id!] = [];
            candidateReasons[sourceTrail.id!] = "selected_trail";
            targetTrailId = result.targetTrailId || sourceTrail.id || "";

            if (autoDiscoveryCandidateTrails.length === 0) {
                show_toast({
                    type: "warning",
                    icon: "triangle-exclamation",
                    text: $_("trail-merge-no-similar-trails"),
                });
                modal.closeModal();
                return;
            }
        } catch (error) {
            console.error("Failed to find similar trails", error);
            const text =
                error instanceof APIError
                    ? translateTrailMergeError(error.message)
                    : $_("trail-merge-suggest-error");
            show_toast({
                type: "error",
                icon: "close",
                text,
            });
            modal.closeModal();
            return;
        } finally {
            loading = false;
        }
    }

    export interface MergeSettings {
        summitLog: boolean;
        photos: boolean;
        comments: boolean;
        delete: boolean;
        tags: boolean;
        likes: boolean;
    }
    
    const settings: MergeSettings = $state({
        ...defaultSettings,
    });

    function mergeTrail() {
        if (!targetTrailId) {
            return;
        }

        const targetTrail = selectableTargets.find((trail) => trail.id === targetTrailId);
        if (!targetTrail) {
            return;
        }

        let sourceTrails = mergeTrailsSelection.filter((trail) => trail.id !== targetTrail.id);
        if (autoDiscoveryMode && autoDiscoverySourceTrail) {
            sourceTrails =
                targetTrail.id === autoDiscoverySourceTrail.id
                    ? autoDiscoveryCandidateTrails.filter((trail) => trail.id !== targetTrail.id)
                    : [autoDiscoverySourceTrail];
        }

        if (sourceTrails.length === 0) {
            return;
        }

        persistSettingsPreference();
        onmerge?.(settings, {
            targetTrail,
            sourceTrails,
        });
        modal.closeModal();
    }

    let warnings = $derived(targetTrailId ? (candidateWarnings[targetTrailId] ?? []) : []);
    let selectedReason = $derived(targetTrailId ? candidateReasons[targetTrailId] : "");
    let targetTrailItems = $derived.by((): SelectItem[] =>
        selectableTargets.map((trail) => ({
            text: trail.name ?? "",
            value: trail.id ?? "",
        })),
    );
</script>

<Modal id="merge-modal" title={modalTitle} size="min-w-md" bind:this={modal}>
    {#snippet content()}
        <div>
            {#if loading}
                <div class="py-6 flex flex-col items-center justify-center gap-3">
                    <div class="spinner light:spinner-dark"></div>
                    <p class="text-sm text-gray-500">
                        {autoDiscoveryMode
                            ? $_("trail-merge-loading-similar-trails")
                            : $_("trail-merge-loading-targets")}
                    </p>
                </div>
            {:else}
                <div class="mb-4">
                    {#if !fixedTargetSelection}
                        <h4 class="font-semibold mb-2">{$_("trail-merge-target")}</h4>
                        <Select
                            name="trail-merge-target"
                            items={targetTrailItems}
                            bind:value={targetTrailId}
                            disabled={selectableTargets.length <= 1}
                        />
                    {/if}
                    {#if autoDiscoveryMode}
                        <p class="text-sm text-gray-500 mt-2">
                            {$_("trail-merge-similar-trails-found", {
                                values: { n: autoDiscoveryCandidateTrails.length },
                            })}
                        </p>
                    {/if}
                    {#if selectedReason}
                        <p class="text-sm text-gray-500 mt-2">{$_(`trail-merge-reason-${selectedReason}`)}</p>
                    {/if}
                </div>

                {#if warnings.length > 0}
                    <div class="mb-4 rounded-xl border border-yellow-500/40 bg-yellow-500/10 p-3">
                        <h4 class="font-semibold mb-2">{$_("trail-merge-warnings-title")}</h4>
                        <ul class="text-sm space-y-1">
                            {#each warnings as warning}
                                <li>{$_(`trail-merge-warning-${warning}`)}</li>
                            {/each}
                        </ul>
                    </div>
                {/if}

            <h4 class="font-semibold mb-2">{$_("copy-include-elements")}</h4>
            <div class="mb-2">
                <input
                    id="include-summit-log-checkbox"
                    type="checkbox"
                    bind:checked={settings.summitLog}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-summit-log-checkbox" class="ms-2 text-sm"
                    >{$_("summit-log", { values: { n: 2 } })}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-photos-checkbox"
                    type="checkbox"
                    bind:checked={settings.photos}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-photos-checkbox" class="ms-2 text-sm"
                    >{$_("photos")}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-comments-checkbox"
                    type="checkbox"
                    bind:checked={settings.comments}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-comments-checkbox" class="ms-2 text-sm"
                    >{$_("comment", { values: { n: 2 } })}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-tags-checkbox"
                    type="checkbox"
                    bind:checked={settings.tags}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-tags-checkbox" class="ms-2 text-sm"
                    >{$_("tags")}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-likes-checkbox"
                    type="checkbox"
                    bind:checked={settings.likes}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-likes-checkbox" class="ms-2 text-sm"
                    >{$_("likes")}</label
                >
            </div>
            <h4 class="font-semibold mt-4 mb-2">{$_("linked-trails")}</h4>
            <div class="mb-2">
                <input
                    id="include-trail-delete-checkbox"
                    type="checkbox"
                    bind:checked={settings.delete}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-trail-delete-checkbox" class="ms-2 text-sm"
                    >{$_("delete-linked-trails")}</label
                >
            </div>
            {/if}
        </div>
    {/snippet}
    {#snippet footer()}
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <label for="remember-merge-settings-checkbox" class="flex items-center gap-2 text-sm">
                <input
                    id="remember-merge-settings-checkbox"
                    type="checkbox"
                    bind:checked={rememberSettings}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <span>{$_("trail-merge-remember-settings-short")}</span>
            </label>
            <div class="flex items-center gap-4">
                <button class="btn-secondary" onclick={() => modal.closeModal()}
                    >{$_("cancel")}</button
                >
                <button
                    class="btn-primary"
                    type="button"
                    disabled={loading || !targetTrailId}
                    onclick={mergeTrail}
                    name="save">{$_("link")}</button
                >
            </div>
        </div>
    {/snippet}</Modal
>
