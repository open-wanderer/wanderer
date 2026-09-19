<script lang="ts">
    import type { RoutingCandidate, RoutingOptions } from "$lib/models/routing";
    import { routingStore } from "$lib/stores/routing_store.svelte";
    import {
        formatDistance,
        formatElevation,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { trailColor } from "$lib/util/trail_color_util";
    import {
        ROUTING_MAX_VARIANTS,
        routingCandidateEngineLabel,
    } from "$lib/util/routing_variant_util";
    import { _ } from "svelte-i18n";
    import { slide } from "svelte/transition";

    interface Props {
        options: RoutingOptions;
        onFindVariants: (desiredVariants?: number) => void | Promise<void>;
        onSelectOriginal: () => void;
        onSelectCandidate: (candidate: RoutingCandidate) => void;
        onApplySelected: () => void;
        onPreviewCandidate: (
            candidate: RoutingCandidate | null | undefined,
        ) => void;
        disabled?: boolean;
        parallelRoutingAvailable?: boolean;
        variantRequestError?: string;
        maxVariants?: number;
    }

    let {
        options = $bindable(),
        onFindVariants,
        onSelectOriginal,
        onSelectCandidate,
        onApplySelected,
        onPreviewCandidate,
        disabled = false,
        parallelRoutingAvailable = false,
        variantRequestError,
        maxVariants = ROUTING_MAX_VARIANTS,
    }: Props = $props();

    let expanded = $state(false);
    let hadCandidateState = false;

    $effect(() => {
        const hasCandidateState =
            routingStore.routeCandidateOrigin !== undefined ||
            routingStore.routeCandidates.length > 0 ||
            routingStore.routeCandidatesLoading ||
            routingStore.routeCandidateErrors.length > 0 ||
            routingStore.routeCandidateWarnings.length > 0 ||
            routingStore.routeCandidateRequestedCount > 0;

        if (hadCandidateState && !hasCandidateState) {
            expanded = false;
        }
        hadCandidateState = hasCandidateState;
    });

    function candidateTitle(index: number) {
        return `${$_("routing-variant")} ${index + 1}`;
    }

    async function findVariants(desiredVariants = options.desiredVariants ?? 1) {
        if (
            disabled ||
            routingStore.routeCandidatesLoading ||
            routingStore.anchors.length < 2 ||
            variantRequestError
        ) {
            return;
        }
        await onFindVariants(Math.min(desiredVariants, maxVariants));
    }

    function selectCandidate(candidate: RoutingCandidate) {
        if (routingStore.routeCandidatesStale) return;
        onSelectCandidate(candidate);
    }

    function selectOriginal() {
        if (routingStore.routeCandidatesStale) return;
        onSelectOriginal();
    }

    function applySelected() {
        onApplySelected();
        onPreviewCandidate(undefined);
        expanded = false;
    }

    function previewCandidate(
        candidate: RoutingCandidate | null | undefined,
    ) {
        onPreviewCandidate(
            routingStore.routeCandidatesStale ? undefined : candidate,
        );
    }

    function candidateColor(index: number) {
        return trailColor(index + 1);
    }

    function failedEngineNames() {
        return Array.from(
            new Set(
                routingStore.routeCandidateErrors
                    .map((error) => error.provider || error.pluginId)
                    .filter((engine): engine is string => Boolean(engine)),
            ),
        );
    }

    let selectedCandidate = $derived(
        routingStore.routeCandidates.find(
            (candidate) => candidate.id === routingStore.selectedRouteCandidateId,
        ),
    );

    async function findMoreVariants() {
        const currentCount = Math.max(
            routingStore.routeCandidateRequestedCount,
            routingStore.routeCandidates.length,
        );
        await findVariants(Math.min(maxVariants, currentCount + 1));
    }

    function canFindMoreVariants() {
        return (
            Math.max(
                routingStore.routeCandidateRequestedCount,
                routingStore.routeCandidates.length,
            ) < maxVariants
        );
    }

    function toggleExpanded() {
        if (disabled) return;
        expanded = !expanded;
        if (
            expanded &&
            !routingStore.routeCandidates.length &&
            !routingStore.routeCandidatesLoading &&
            !variantRequestError
        ) {
            void findVariants();
        }
    }

</script>

<section class="w-full">
    <div
        class="flex w-full items-center rounded-lg border border-input-border transition-colors"
        class:rounded-b-none={expanded}
        class:opacity-60={disabled}
        title={disabled ? $_("routing-original-route-calculating") : undefined}
    >
        <div class="flex min-w-0 flex-1 items-center gap-3 px-4 py-3">
            <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-input-background text-content"
            >
                <i class="fa fa-code-branch" aria-hidden="true"></i>
            </span>
            <span class="min-w-0 flex-1 font-medium">{$_("routing-variants")}</span>
        </div>
        {#if expanded}
            <button
                class="btn-icon shrink-0"
                type="button"
                title={variantRequestError ?? $_("routing-find-variants")}
                aria-label={$_("routing-find-variants")}
                disabled={disabled ||
                    routingStore.routeCandidatesLoading ||
                    routingStore.anchors.length < 2 ||
                    !!variantRequestError}
                onclick={() => findVariants()}
            >
                {#if routingStore.routeCandidatesLoading}
                    <span class="spinner spinner-small" aria-hidden="true"></span>
                {:else}
                    <i class="fa fa-rotate-right" aria-hidden="true"></i>
                {/if}
            </button>
        {/if}
        <button
            type="button"
            class="btn-icon mr-2 shrink-0"
            aria-controls="route-variants-panel"
            aria-expanded={expanded}
            aria-label={$_("routing-variants")}
            disabled={disabled}
            onclick={toggleExpanded}
        >
            <i
                class="fa fa-chevron-{expanded ? 'up' : 'down'} text-xs text-gray-500"
                aria-hidden="true"
            ></i>
        </button>
    </div>

    {#if expanded}
        <div
            id="route-variants-panel"
            class="space-y-3 rounded-b-lg border-x border-b border-input-border p-3"
            transition:slide
        >
            {#if !parallelRoutingAvailable}
                <p class="text-xs text-gray-500">
                    {$_("routing-parallel-disabled")}
                </p>
            {/if}
            {#if variantRequestError}
                <p class="text-xs text-orange-500">{variantRequestError}</p>
            {/if}
            {#if routingStore.routeCandidatesStale}
                <p class="text-xs text-orange-500">
                    {$_("routing-variants-stale")}
                </p>
            {/if}
            {#if routingStore.routeCandidateErrors.length}
                <div class="space-y-1 text-xs text-orange-500">
                    <p>
                        {$_("routing-partial-engine-errors", {
                            values: { n: routingStore.routeCandidateErrors.length },
                        })}
                    </p>
                    {#if failedEngineNames().length}
                        <p class="text-gray-500">
                            {$_("routing-partial-engine-names", {
                                values: { engines: failedEngineNames().join(", ") },
                            })}
                        </p>
                    {/if}
                </div>
            {/if}
            {#if routingStore.routeCandidateWarnings.includes("routing_variants_reduced_for_short_route")}
                <p class="text-xs text-gray-500">
                    {$_("routing-short-route-variants-reduced")}
                </p>
            {/if}
            {#if routingStore.routeCandidateWarnings.includes("routing_mode_fallback")}
                <p class="text-xs text-orange-500">
                    {$_("routing-mode-fallback")}
                </p>
            {/if}
            {#if routingStore.routeCandidateWarnings.includes("routing_profile_preparation_failed")}
                <p class="text-xs text-orange-500">
                    {$_("routing-profile-preparation-failed")}
                </p>
            {/if}
            {#if routingStore.routeCandidateWarnings.includes("routing_parallel_engines_reduced_for_fanout")}
                <p class="text-xs text-gray-500">
                    {$_("routing-parallel-engines-reduced")}
                </p>
            {/if}
            {#if routingStore.routeCandidateWarnings.includes("routing_variants_similar_to_original_removed")}
                <p class="text-xs text-gray-500">
                    {$_("routing-similar-variants-removed")}
                </p>
            {/if}
            {#if routingStore.routeCandidateWarnings.includes("routing_variants_fewer_than_requested")}
                <p class="text-xs text-gray-500">
                    {$_("routing-fewer-variants-found", {
                        values: {
                            count: routingStore.routeCandidates.length,
                            requested: routingStore.routeCandidateRequestedCount,
                        },
                    })}
                </p>
            {/if}
            {#if routingStore.routeCandidateOrigin || routingStore.routeCandidates.length}
                <div class="max-h-96 space-y-2 overflow-y-auto">
                    {#if routingStore.routeCandidateOrigin}
                        <article
                            class="rounded-lg border border-input-border transition-colors"
                            class:opacity-60={routingStore.routeCandidatesStale}
                            class:border-primary={routingStore.selectedRouteCandidateId === null}
                            class:bg-input-background={routingStore.selectedRouteCandidateId === null}
                            onmouseenter={() => previewCandidate(null)}
                            onmouseleave={() => previewCandidate(undefined)}
                            onfocusin={() => previewCandidate(null)}
                            onfocusout={() => previewCandidate(undefined)}
                        >
                            <button
                                type="button"
                                class="w-full rounded-lg p-3 text-left hover:bg-secondary-hover focus:ring-4 focus:ring-input-ring"
                                disabled={routingStore.routeCandidatesStale}
                                onclick={selectOriginal}
                            >
                                <span class="flex items-center gap-3">
                                    <i
                                        class="fa fa-route w-5 shrink-0 text-center text-lg"
                                        style:color={trailColor(0)}
                                        aria-hidden="true"
                                    ></i>
                                    <strong class="flex-1">
                                        {$_("routing-original-route")}
                                    </strong>
                                </span>
                                <span
                                    class="mt-2 grid grid-cols-3 gap-2 pl-8 text-xs text-gray-500"
                                >
                                    <span class="whitespace-nowrap">
                                        <i class="fa fa-left-right mr-1" aria-hidden="true"></i>
                                        {formatDistance(
                                            routingStore.routeCandidateOrigin.route.features.distance,
                                        )}
                                    </span>
                                    <span class="whitespace-nowrap">
                                        <i class="fa fa-clock mr-1" aria-hidden="true"></i>
                                        {formatTimeHHMM(
                                            routingStore.routeCandidateOrigin.route.features.duration /
                                                1000,
                                        )}
                                    </span>
                                    <span class="whitespace-nowrap">
                                        <i class="fa fa-arrow-trend-up mr-1" aria-hidden="true"></i>
                                        {formatElevation(
                                            routingStore.routeCandidateOrigin.route.features
                                                .elevationGain ?? 0,
                                        )}
                                    </span>
                                </span>
                            </button>
                        </article>
                    {/if}
                    {#each routingStore.routeCandidates as candidate, candidateIndex}
                        {@const engineLabel = routingCandidateEngineLabel(candidate)}
                        <article
                            class="rounded-lg border border-input-border transition-colors"
                            class:opacity-60={routingStore.routeCandidatesStale}
                            class:border-primary={routingStore.selectedRouteCandidateId === candidate.id}
                            class:bg-input-background={routingStore.selectedRouteCandidateId === candidate.id}
                            onmouseenter={() => previewCandidate(candidate)}
                            onmouseleave={() => previewCandidate(undefined)}
                            onfocusin={() => previewCandidate(candidate)}
                            onfocusout={() => previewCandidate(undefined)}
                        >
                            <button
                                type="button"
                                class="w-full rounded-lg p-3 text-left hover:bg-secondary-hover focus:ring-4 focus:ring-input-ring"
                                aria-label={`${$_("select")}: ${candidateTitle(candidateIndex)}`}
                                disabled={routingStore.routeCandidatesStale}
                                onclick={() => selectCandidate(candidate)}
                            >
                                <span class="flex items-center gap-3">
                                    <i
                                        class="fa fa-route w-5 shrink-0 text-center text-lg"
                                        style:color={candidateColor(candidateIndex)}
                                        aria-hidden="true"
                                    ></i>
                                    <span class="min-w-0 flex-1">
                                        <strong class="block">
                                            {candidateTitle(candidateIndex)}
                                        </strong>
                                        {#if engineLabel}
                                            <span class="block truncate text-xs font-normal text-gray-500">
                                                {engineLabel}
                                            </span>
                                        {/if}
                                    </span>
                                </span>
                                <span
                                    class="mt-2 grid grid-cols-3 gap-2 pl-8 text-xs text-gray-500"
                                >
                                    <span class="whitespace-nowrap">
                                        <i class="fa fa-left-right mr-1" aria-hidden="true"></i>
                                        {formatDistance(candidate.summary.distance)}
                                    </span>
                                    <span class="whitespace-nowrap">
                                        <i class="fa fa-clock mr-1" aria-hidden="true"></i>
                                        {formatTimeHHMM(candidate.summary.duration)}
                                    </span>
                                    <span class="whitespace-nowrap">
                                        <i class="fa fa-arrow-trend-up mr-1" aria-hidden="true"></i>
                                        {formatElevation(candidate.summary.elevationGain ?? 0)}
                                    </span>
                                </span>
                            </button>
                        </article>
                    {/each}
                </div>
            {/if}
            {#if selectedCandidate}
                <button
                    class="btn-primary w-full"
                    type="button"
                    disabled={routingStore.routeCandidatesStale}
                    onclick={applySelected}
                >
                    {$_("routing-apply-variant")}
                </button>
            {/if}
            {#if routingStore.routeCandidates.length && canFindMoreVariants()}
                <button
                    class="btn-secondary w-full"
                    type="button"
                    disabled={routingStore.routeCandidatesLoading ||
                        routingStore.routeCandidatesStale ||
                        !!variantRequestError}
                    onclick={findMoreVariants}
                >
                    {$_("routing-find-more-variants")}
                </button>
            {/if}
        </div>
    {/if}
</section>
