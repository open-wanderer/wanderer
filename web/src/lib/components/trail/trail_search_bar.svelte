<script lang="ts">
    import SearchFilterSummary, { type SearchFilterChip } from "../base/search_filter_summary.svelte";
    import type { ActorSearchResult } from "$lib/models/activitypub/actor";
    import type { Category } from "$lib/models/category";
    import type { Subcategory } from "$lib/models/subcategory";
    import { searchActorById, type LocationSearchResult } from "$lib/stores/search_store";
    import type { TrailFilter } from "$lib/models/trail";
    import { normalizeTrailSearchFilterSections, trailSearchFilters, type TrailSearchFilterId, type TrailSearchFilterSections } from "$lib/models/trail_search_settings";
    import { categoryPreferences } from "$lib/stores/category_preference_store";
    import { subcategoryPreferences } from "$lib/stores/subcategory_preference_store";
    import { currentUser } from "$lib/stores/user_store";
    import {
        displayCategoryIcon, displayCategoryName, displaySubcategoryLabel,
        displaySubcategoryIcon, displaySubcategoryBadgeIcon, preferenceForCategory, sortedCategoriesByPreference,
        sortedSubcategoriesByPreference, subcategoryVisible,
    } from "$lib/util/category_util";
    import { noSubcategoryFilterCategory, noSubcategoryFilterValue } from "$lib/util/trail_filter_util";
    import type { TrailSearchBounds } from "$lib/util/trail_search_util";
    import { deepEqual } from "$lib/util/deep_util";
    import { fromDraft, toDraft, type TrailSearchDraft } from "$lib/util/trail_search_draft";
    import { formatDistance, formatElevation } from "$lib/util/format_util";
    import {
        consumeTrailSearchShortcut, trailSearchAuthorHandle,
        type TrailSearchShortcut, type TrailSearchShortcutChoice,
    } from "$lib/util/trail_search_shortcuts";
    import { onMount, tick, untrack } from "svelte";
    import { _, locale } from "svelte-i18n";
    import TrailSearchField from "./trail_search_field.svelte";
    import TrailSearchRange from "./trail_search_range.svelte";
    import TrailSearchLocation from "./trail_search_location.svelte";
    import TrailSearchShortcuts from "./trail_search_shortcuts.svelte";
    import TrailSearchSuggestions from "./trail_search_suggestions.svelte";

    interface Props {
        categories: Category[];
        subcategories: Subcategory[];
        filter: TrailFilter;
        defaults: TrailFilter;
        inlineControls?: boolean;
        layout?: "overlay" | "push" | "sidebar";
        filterPanelTarget?: HTMLDivElement;
        bounds?: TrailSearchBounds;
        filterSections?: TrailSearchFilterSections;
        onclearbounds?: () => void;
        onapply: (filter: TrailFilter, resetBounds?: boolean) => void;
    }
    let { categories, subcategories, filter = $bindable(), defaults, onapply, inlineControls = false, layout = "overlay", filterPanelTarget, bounds, filterSections, onclearbounds }: Props = $props();
    const id = $props.id();
    let sections = $derived(normalizeTrailSearchFilterSections(filterSections));
    let availableEditors = $derived(trailSearchFilters.filter(editor => !("requiresUser" in editor) || Boolean($currentUser)));
    let generalEditors = $derived(availableEditors.filter(editor => sections[editor.id] === "general"));
    let advancedEditors = $derived(availableEditors.filter(editor => sections[editor.id] === "advanced"));
    let draft = $state<TrailSearchDraft>(toDraft(filter));
    let expanded = $state(false);
    let moreExpanded = $state(false);
    let activeEditor = $state<TrailSearchFilterId | null>("activity");
    let activeAdvancedEditor = $state<TrailSearchFilterId | null>(null);
    let activeCategoryId = $state<string | null>(null);
    let panelMaxHeight = $state(0);
    let searchRoot: HTMLDivElement;
    let searchHeading: HTMLDivElement;
    let searchPanel = $state<HTMLDivElement>();
    let searchInput: HTMLInputElement;
    let shortcuts = $state<TrailSearchShortcuts>();
    let suggestions = $state<TrailSearchSuggestions>();
    let queryCaret = $state(0);
    let querySelectionEnd = $state(0);
    let authorLabels = $state<Record<string, string>>({});
    const authorLookups = new Map<string, AbortController>();
    let restoringFocus = false;
    let updateTimer: ReturnType<typeof setTimeout> | undefined;
    let composing = $state(false);
    let resetBounds = $state(false);
    let pinned = $derived(layout === "sidebar");
    let resetEnabled = $derived(Boolean(expanded || (expanded ? draft : filter).q || summary(expanded ? draft : filter).length || (bounds && onclearbounds)));
    let categoryItems = $derived(sortedCategoriesByPreference(categories, $categoryPreferences, $locale)
        .filter((item) => preferenceForCategory($categoryPreferences, item.id)?.visible !== false || draft.category.includes(item.id)));
    let refinableCategoryIds = $derived(draft.category.filter(canRefineCategory));
    let selectedCategory = $derived(categories.find((item) => item.id === activeCategoryId));
    let activeSubcategories = $derived(sortedSubcategoriesByPreference(subcategories.filter((item) =>
        item.category === activeCategoryId && (subcategoryVisible(item.id, $subcategoryPreferences) || draft.subcategory.includes(item.id))), $subcategoryPreferences, $locale));
    let appliedChips = $derived<SearchFilterChip[]>([
        ...(bounds && onclearbounds ? [{ key: "bounds", label: $_("trail-search-map-area"), text: "", icon: "fa-map" }] : []),
        ...summary(filter).map(chip => ({ ...chip, icon: appliedIcon(chip.key) })),
    ]);
    let advancedCount = $derived(summary(draft).filter(chip => advancedEditors.some(editor =>
        editor.id === (chip.key.startsWith("category:") || chip.key.startsWith("subcategory:") ? "activity" : chip.key))).length);

    $effect(() => {
        const authors = [...new Set([filter.author, draft.author].filter((id): id is string => Boolean(id)))];
        const missing = untrack(() => authors.filter(id => !authorLabels[id]));
        for (const authorId of missing) {
            if (authorLookups.has(authorId)) continue;
            const controller = new AbortController();
            authorLookups.set(authorId, controller);
            void searchActorById(authorId, fetch, controller.signal).then(author => {
                if (author && !controller.signal.aborted) rememberAuthor(author);
            }).catch(() => {}).finally(() => authorLookups.delete(authorId));
        }
    });

    function rememberAuthor(author: ActorSearchResult) {
        authorLabels[author.id] = trailSearchAuthorHandle(author);
    }

    $effect(() => {
        if (!generalEditors.some(editor => editor.id === activeEditor)) activeEditor = generalEditors[0]?.id ?? null;
        if (!advancedEditors.some(editor => editor.id === activeAdvancedEditor)) activeAdvancedEditor = advancedEditors[0]?.id ?? null;
    });

    $effect(() => {
        if (!refinableCategoryIds.includes(activeCategoryId ?? "")) {
            activeCategoryId = refinableCategoryIds[0] ?? null;
        }
    });

    $effect(() => {
        if (pinned) untrack(openSearch);
    });

    $effect(() => {
        if (!expanded || composing) return;
        const current = toDraft(draft);
        const clearBounds = resetBounds;
        if (untrack(() => deepEqual(current, toDraft(filter))) && !clearBounds) return;
        const timer = setTimeout(() => publishFilter(current, clearBounds), 350);
        updateTimer = timer;
        return () => clearTimeout(timer);
    });

    onMount(() => {
        const viewport = window.visualViewport;
        const headingSize = new ResizeObserver(updatePanelHeight);
        headingSize.observe(searchHeading);
        headingSize.observe(searchRoot);
        viewport?.addEventListener("resize", updatePanelHeight);
        viewport?.addEventListener("scroll", updatePanelHeight);
        return () => {
            for (const controller of authorLookups.values()) controller.abort();
            headingSize.disconnect();
            viewport?.removeEventListener("resize", updatePanelHeight);
            viewport?.removeEventListener("scroll", updatePanelHeight);
        };
    });

    function hasPlace(value: TrailSearchDraft) { return Number.isFinite(value.near.lat) && Number.isFinite(value.near.lon); }
    function locationName(value: TrailSearchDraft) {
        return value.near.name || `${value.near.lat?.toFixed(3)}, ${value.near.lon?.toFixed(3)}`;
    }
    function difficultyName(value: number) { return $_(["easy", "moderate", "difficult"][value]); }
    function sidebarHeadingIcon(editor: TrailSearchFilterId, fallback: string) {
        switch (editor) {
            case "activity": return null;
            case "distance": return "fa-left-right";
            case "difficulty": return "fa-gauge";
            case "completed": return draft.completed === undefined
                ? null
                : draft.completed ? "fa-flag-checkered" : "fa-compass-drafting";
            default: return fallback;
        }
    }
    function isActive(editor: TrailSearchFilterId, value: TrailSearchDraft) {
        switch (editor) {
            case "activity": return value.category.length > 0 || value.subcategory.length > 0;
            case "location": return hasPlace(value);
            case "distance": return value.distanceMin > 0 || value.distanceMax < value.distanceLimit;
            case "difficulty": return value.difficulty.length > 0 && value.difficulty.length < 3;
            case "tags": return value.tags.length > 0;
            case "author": return Boolean(value.author);
            case "visibility": return value.public !== defaults.public || value.private !== defaults.private || value.shared !== defaults.shared;
            case "completed": return value.completed !== undefined;
            case "gain": return value.elevationGainMin > 0 || value.elevationGainMax < value.elevationGainLimit;
            case "loss": return value.elevationLossMin > 0 || value.elevationLossMax < value.elevationLossLimit;
            case "date": return Boolean(value.startDate || value.endDate);
            case "liked": return Boolean(value.liked);
        }
    }
    function chipLabel(editor: TrailSearchFilterId, key: string) {
        if (!isActive(editor, draft)) return $_(key);
        switch (editor) {
            case "activity": {
                const category = categories.find((item) => item.id === draft.category[0]);
                return draft.category.length === 1 && category ? displayCategoryName(category, $locale) : `${$_(key)} · ${draft.category.length}`;
            }
            case "location": return `${locationName(draft)} · ${formatDistance(draft.near.radius)}`;
            case "distance": return `${formatDistance(draft.distanceMin)} – ${formatDistance(draft.distanceMax)}${draft.distanceMax >= draft.distanceLimit ? "+" : ""}`;
            case "difficulty": return draft.difficulty.map(difficultyName).join(", ");
            default: return $_(key);
        }
    }
    function openSearch() {
        if (expanded) return;
        draft = toDraft(filter);
        queryCaret = querySelectionEnd = draft.q.length;
        resetBounds = false;
        activeCategoryId = filter.category[0] ?? null;
        expanded = true;
        updatePanelHeight();
    }
    function focusInput() {
        restoringFocus = true;
        searchInput.focus();
        restoringFocus = false;
    }
    function publishFilter(next = toDraft(draft), clearBounds = resetBounds) {
        clearTimeout(updateTimer);
        const applied = fromDraft(next, filter);
        if (deepEqual(applied, filter) && !clearBounds) return;
        filter = applied;
        resetBounds = false;
        onapply(fromDraft(next, applied), clearBounds);
    }
    function dismiss(restoreFocus = false) {
        if (!expanded) return;
        publishFilter();
        if (!pinned) expanded = false;
        if (restoreFocus) focusInput();
    }
    function selectSuggestedLocation(location: LocationSearchResult) {
        // Capture the query as an explicit place filter, just like selecting
        // a place in the location editor, without changing the result view.
        draft.q = "";
        draft.near = { ...draft.near, lat: location.lat, lon: location.lon, name: location.name };
        resetBounds = true;
        dismiss();
    }
    function containsSearch(node: Node) {
        return searchRoot.contains(node) || Boolean(searchPanel?.contains(node));
    }
    function placeFilterPanel(node: HTMLDivElement, target: HTMLDivElement | undefined) {
        const anchor = document.createComment("search-filter-panel");
        node.before(anchor);
        function move(next: HTMLDivElement | undefined) {
            if (next) next.append(node);
            else anchor.after(node);
            updatePanelHeight();
        }
        move(target);
        return {
            update: move,
            destroy() { node.remove(); anchor.remove(); },
        };
    }
    function updatePanelHeight() {
        if (pinned || !expanded || !searchHeading) return;
        const viewport = window.visualViewport;
        const top = viewport?.offsetTop ?? 0;
        const bottom = top + (viewport?.height ?? window.innerHeight);
        const summary = layout === "push" ? (searchRoot.querySelector<HTMLElement>(".applied-filters:not(.inline)")?.offsetHeight ?? 0) : 0;
        const panelTop = searchHeading.getBoundingClientRect().bottom;
        let available = bottom - panelTop - 16 - summary;
        const room = searchHeading.getBoundingClientRect().top - top - 12;
        if (available < 240 && room > 0) {
            window.scrollBy({ top: Math.min(240 - available, room), behavior: "instant" });
            available = bottom - searchHeading.getBoundingClientRect().bottom - 16 - summary;
        }
        panelMaxHeight = Math.max(0, available);
    }
    function handleOutsidePointer(event: PointerEvent) {
        if (expanded && event.target instanceof Node && !containsSearch(event.target)) dismiss();
    }
    function handleFocusOut(event: FocusEvent) {
        if (!expanded || !(event.relatedTarget instanceof Node)) return;
        if (!containsSearch(event.relatedTarget)) dismiss();
    }
    function handleKeydown(event: KeyboardEvent) {
        if (event.isComposing || composing) return;
        if (event.key === "Escape" && expanded) {
            if (pinned && (!(event.target instanceof Node) || !containsSearch(event.target))) return;
            event.preventDefault();
            event.stopPropagation();
            dismiss(true);
            return;
        }
        if (!(event.target instanceof Node) || !containsSearch(event.target)) return;
        if (shortcuts?.handleKeydown(event, event.target === searchInput)) return;
        if (suggestions?.handleKeydown(event, event.target === searchInput)) return;
        if (event.key === "ArrowDown" && event.target === searchInput && !expanded) {
            event.preventDefault(); openSearch();
        }
    }
    function updateQuery(event: Event) {
        const value = (event.currentTarget as HTMLInputElement).value;
        openSearch();
        draft.q = value;
        updateQuerySelection(event);
    }
    function updateQuerySelection(event: Event) {
        const input = event.currentTarget as HTMLInputElement;
        queryCaret = input.selectionStart ?? input.value.length;
        querySelectionEnd = input.selectionEnd ?? queryCaret;
    }
    function captureShortcut(choice: TrailSearchShortcutChoice, shortcut: TrailSearchShortcut) {
        if (choice.kind === "author") {
            draft.author = choice.author.id;
            rememberAuthor(choice.author);
        } else if (choice.kind === "tag") {
            if (!draft.tags.includes(choice.tag.name)) draft.tags = [...draft.tags, choice.tag.name];
        }
        const updated = consumeTrailSearchShortcut(draft.q, shortcut);
        draft.q = updated.value;
        queryCaret = querySelectionEnd = updated.caret;
        void tick().then(() => {
            focusInput();
            searchInput.setSelectionRange(updated.caret, updated.caret);
        });
    }
    function removeShortcut(choice: TrailSearchShortcutChoice) {
        if (choice.kind === "author") draft.author = "";
        else if (choice.kind === "tag") draft.tags = draft.tags.filter(tag => tag !== choice.tag.name);
    }
    function canRefineCategory(categoryId: string) {
        return subcategories.some(item => item.category === categoryId
            && (subcategoryVisible(item.id, $subcategoryPreferences) || draft.subcategory.includes(item.id)));
    }
    function toggleCategory(categoryId: string) {
        if (draft.category.includes(categoryId)) {
            draft.category = draft.category.filter((item) => item !== categoryId);
            const ids = subcategories.filter((item) => item.category === categoryId).map((item) => item.id);
            draft.subcategory = draft.subcategory.filter((item) => !ids.includes(item) && noSubcategoryFilterCategory(item) !== categoryId);
        } else {
            draft.category = [...draft.category, categoryId];
            if (canRefineCategory(categoryId)) activeCategoryId = categoryId;
        }
    }
    function selectedSubcategories() {
        const ids = activeSubcategories.map((item) => item.id);
        return draft.subcategory.filter((item) => ids.includes(item) || noSubcategoryFilterCategory(item) === activeCategoryId);
    }
    function clearSubcategories() {
        const ids = subcategories.filter((item) => item.category === activeCategoryId).map((item) => item.id);
        draft.subcategory = draft.subcategory.filter((item) => !ids.includes(item) && noSubcategoryFilterCategory(item) !== activeCategoryId);
    }
    function toggleSubcategory(subcategoryId: string) {
        draft.subcategory = draft.subcategory.includes(subcategoryId) ? draft.subcategory.filter((item) => item !== subcategoryId) : [...draft.subcategory, subcategoryId];
    }
    function applySearch(event: SubmitEvent) {
        event.preventDefault();
        if (composing) return;
        dismiss(true);
    }
    function resetSearch() {
        if (!expanded) draft = toDraft(filter);
        resetDraft();
        publishFilter();
    }
    function resetDraft() {
        resetBounds = true;
        draft = { ...toDraft(defaults), q: "", category: [], subcategory: [], author: "", near: { radius: defaults.near.radius } };
        activeCategoryId = null;
    }
    function compactRange(min: number, max: number, limit: number, format: (value: number) => string) {
        const shorter = (value: number) => format(value).replace(/\.0+(?=\s)/, "");
        if (min === 0) return `≤ ${shorter(max)}`;
        if (max >= limit) return `≥ ${shorter(min)}`;
        const lower = shorter(min);
        const upper = shorter(max);
        const suffix = upper.slice(upper.lastIndexOf(" "));
        return `${lower.endsWith(suffix) ? lower.slice(0, -suffix.length) : lower}–${upper}`;
    }
    function compactDates(value: TrailSearchDraft) {
        const date = (value: string) => {
            const parsed = new Date(`${value}T00:00:00Z`);
            return Number.isFinite(parsed.getTime())
                ? new Intl.DateTimeFormat($locale ?? "en", { dateStyle: "short", timeZone: "UTC" }).format(parsed)
                : value;
        };
        if (!value.startDate) return `≤ ${date(value.endDate!)}`;
        if (!value.endDate) return `≥ ${date(value.startDate)}`;
        return `${date(value.startDate)}–${date(value.endDate)}`;
    }
    function visibilityText(value: TrailSearchDraft) {
        return [value.private && $_("private"), value.public && $_("public"), value.shared && $_("shared")].filter(Boolean).join(", ") || "–";
    }
    function summary(value: TrailSearchDraft) {
        const chips: { key: string; label: string; text?: string }[] = [];
        for (const categoryId of value.category) {
            const category = categories.find((item) => item.id === categoryId);
            if (category) {
                const subs = subcategories.filter((item) => item.category === categoryId && value.subcategory.includes(item.id)).map((item) => displaySubcategoryLabel(item, $locale));
                if (value.subcategory.includes(noSubcategoryFilterValue(categoryId))) subs.push($_("no-subcategory"));
                chips.push({ key: `category:${categoryId}`, label: [displayCategoryName(category, $locale), ...subs].join(" / ") });
            }
        }
        // Links can select a subcategory without a parent category.
        for (const sub of subcategories.filter((item) => value.subcategory.includes(item.id) && !value.category.includes(item.category))) {
            chips.push({ key: `subcategory:${sub.id}`, label: displaySubcategoryLabel(sub, $locale) });
        }
        if (hasPlace(value)) chips.push({ key: "location", label: `${locationName(value)} · ${formatDistance(value.near.radius)}` });
        if (isActive("distance", value)) chips.push({ key: "distance", label: `${$_("distance")}: ${formatDistance(value.distanceMin)} – ${formatDistance(value.distanceMax)}`, text: compactRange(value.distanceMin, value.distanceMax, value.distanceLimit, formatDistance) });
        if (isActive("difficulty", value)) chips.push({ key: "difficulty", label: value.difficulty.map(difficultyName).join(", ") });
        if (value.tags.length) chips.push({ key: "tags", label: `${$_("tags")}: ${value.tags.join(", ")}`, text: value.tags.join(", ") });
        if (value.author) chips.push({ key: "author", label: authorLabels[value.author] ? `${$_("author")}: ${authorLabels[value.author]}` : $_("author"), text: authorLabels[value.author] ?? $_("author") });
        if (value.public !== defaults.public || value.private !== defaults.private || value.shared !== defaults.shared) chips.push({ key: "visibility", label: `${$_("visibilty-status")}: ${visibilityText(value)}`, text: visibilityText(value) });
        if (value.elevationGainMin > 0 || value.elevationGainMax < value.elevationGainLimit) chips.push({ key: "gain", label: `${$_("elevation-gain")}: ${formatElevation(value.elevationGainMin)} – ${formatElevation(value.elevationGainMax)}`, text: compactRange(value.elevationGainMin, value.elevationGainMax, value.elevationGainLimit, formatElevation) });
        if (value.elevationLossMin > 0 || value.elevationLossMax < value.elevationLossLimit) chips.push({ key: "loss", label: `${$_("elevation-loss")}: ${formatElevation(value.elevationLossMin)} – ${formatElevation(value.elevationLossMax)}`, text: compactRange(value.elevationLossMin, value.elevationLossMax, value.elevationLossLimit, formatElevation) });
        if (value.startDate || value.endDate) chips.push({ key: "date", label: `${$_("date")}: ${value.startDate ?? "…"} – ${value.endDate ?? "…"}`, text: compactDates(value) });
        if (value.liked) chips.push({ key: "liked", label: $_("liked"), text: "" });
        if (value.completed !== undefined) chips.push({ key: "completed", label: value.completed ? $_("completed") : $_("not-completed") });
        return chips;
    }
    function appliedIcon(key: string) {
        if (key.startsWith("category:")) {
            return displayCategoryIcon(categories.find(category => category.id === key.slice(9)));
        }
        if (key.startsWith("subcategory:")) {
            const subcategory = subcategories.find(subcategory => subcategory.id === key.slice(12));
            return displaySubcategoryIcon(subcategory, categories.find(category => category.id === subcategory?.category));
        }
        return trailSearchFilters.find(editor => editor.id === key)?.icon ?? "fa-filter";
    }
    function removeApplied(key: string) {
        if (key === "bounds") { onclearbounds?.(); return; }
        const next = toDraft(expanded ? draft : filter);
        if (key.startsWith("category:")) {
            const categoryId = key.slice(9);
            next.category = next.category.filter((item) => item !== categoryId);
            const ids = subcategories.filter((item) => item.category === categoryId).map((item) => item.id);
            next.subcategory = next.subcategory.filter((item) => !ids.includes(item) && noSubcategoryFilterCategory(item) !== categoryId);
        } else if (key.startsWith("subcategory:")) next.subcategory = next.subcategory.filter((item) => item !== key.slice(12));
        else switch (key) {
            case "location": next.near = { radius: defaults.near.radius }; break;
            case "distance": next.distanceMin = 0; next.distanceMax = next.distanceLimit; break;
            case "difficulty": next.difficulty = [...defaults.difficulty]; break;
            case "tags": next.tags = []; break;
            case "author": next.author = ""; break;
            case "visibility": next.public = defaults.public; next.private = defaults.private; next.shared = defaults.shared; break;
            case "gain": next.elevationGainMin = 0; next.elevationGainMax = next.elevationGainLimit; break;
            case "loss": next.elevationLossMin = 0; next.elevationLossMax = next.elevationLossLimit; break;
            case "date": next.startDate = undefined; next.endDate = undefined; break;
            case "liked": next.liked = false; break;
            case "completed": next.completed = undefined; break;
        }
        publishFilter(next);
        if (expanded) {
            draft = toDraft(filter);
            if (!draft.category.includes(activeCategoryId ?? "")) activeCategoryId = draft.category[0] ?? null;
        }
    }
</script>

{#snippet filterEditor(editor: TrailSearchFilterId)}
    {#if editor === "activity"}
        <p class="editor-help">{$_("trail-search-activity-help")}</p>
        <div class="activity-grid">
            {#each categoryItems as category}
                {@const selected = draft.category.includes(category.id)}
                <button type="button" class="activity-option" class:chosen={selected} aria-pressed={selected} onclick={() => toggleCategory(category.id)}>
                    <i class={`fa ${displayCategoryIcon(category)} activity-icon`} aria-hidden="true"></i>
                    <span>{displayCategoryName(category, $locale)}</span>
                    {#if selected}<i class="fa fa-check selection-check" aria-hidden="true"></i>{/if}
                </button>
            {/each}
        </div>
        {#if refinableCategoryIds.length}
            <div class="refinement">
                <div class="refinement-heading"><span>{$_("trail-search-refine")}</span>
                    <div class="category-switch" role="group" aria-label={$_("trail-search-refine")}>
                        {#each refinableCategoryIds as categoryId}
                            {@const category = categories.find((item) => item.id === categoryId)}
                            {#if category}<button type="button" class:current={activeCategoryId === category.id} aria-pressed={activeCategoryId === category.id} onclick={() => activeCategoryId = category.id}>{displayCategoryName(category, $locale)}</button>{/if}
                        {/each}
                    </div>
                </div>
                {#if selectedCategory && activeSubcategories.length}
                    {@const allSelected = selectedSubcategories().length === 0}
                    {@const noneSelected = draft.subcategory.includes(noSubcategoryFilterValue(selectedCategory.id))}
                    <div class="choice-row">
                        <button type="button" class="choice subcategory-choice" class:chosen={allSelected} aria-pressed={allSelected} onclick={clearSubcategories}>
                            <span class="subcategory-icon" aria-hidden="true"><i class="fa fa-layer-group"></i></span>
                            <span>{$_("trail-search-all-subcategories")}</span>
                            {#if allSelected}<i class="fa fa-check selection-check" aria-hidden="true"></i>{/if}
                        </button>
                        <button type="button" class="choice subcategory-choice" class:chosen={noneSelected} aria-pressed={noneSelected} onclick={() => toggleSubcategory(noSubcategoryFilterValue(selectedCategory!.id))}>
                            <span class="subcategory-icon" aria-hidden="true"><i class={`fa ${displayCategoryIcon(selectedCategory)}`}></i><i class="fa-regular fa-circle subcategory-badge"></i></span>
                            <span>{$_("no-subcategory")}</span>
                            {#if noneSelected}<i class="fa fa-check selection-check" aria-hidden="true"></i>{/if}
                        </button>
                        {#each activeSubcategories as subcategory}
                            {@const selected = draft.subcategory.includes(subcategory.id)}
                            {@const badgeIcon = displaySubcategoryBadgeIcon(subcategory)}
                            <button type="button" class="choice subcategory-choice" class:chosen={selected} aria-pressed={selected} onclick={() => toggleSubcategory(subcategory.id)}>
                                <span class="subcategory-icon" aria-hidden="true">
                                    <i class={`fa ${displaySubcategoryIcon(subcategory, selectedCategory)}`}></i>
                                    {#if badgeIcon}<i class={`fa ${badgeIcon} subcategory-badge`}></i>{/if}
                                </span>
                                <span>{displaySubcategoryLabel(subcategory, $locale)}</span>
                                {#if selected}<i class="fa fa-check selection-check" aria-hidden="true"></i>{/if}
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}
    {:else if editor === "location"}
        <TrailSearchLocation bind:near={draft.near} />
    {:else if editor === "distance"}
        <TrailSearchRange label={$_("distance")} bind:min={draft.distanceMin} bind:max={draft.distanceMax} limit={draft.distanceLimit} formatValue={formatDistance} />
    {:else}
        <TrailSearchField field={editor} bind:filter={draft} authorLabel={authorLabels[draft.author ?? ""]} onauthorselect={rememberAuthor} />
    {/if}
{/snippet}

{#snippet filterGroup(editors: typeof availableEditors, active: TrailSearchFilterId | null, select: (editor: TrailSearchFilterId) => void, editorId: string, label: string, includeMore = false)}
    <p class="filter-caption">{label}</p>
    <div class="filter-chips" role="group" aria-label={label}>
        {#each editors as editor}
            <button type="button" class="filter-chip" class:editing={active === editor.id} class:selected={isActive(editor.id, draft)} aria-pressed={active === editor.id} aria-controls={editorId} onclick={() => select(editor.id)}>
                <i class={`fa ${editor.icon}`} aria-hidden="true"></i><span>{chipLabel(editor.id, editor.label)}</span>
                {#if isActive(editor.id, draft)}<i class="fa fa-check chip-check" aria-hidden="true"></i>{/if}
            </button>
        {/each}
        {#if includeMore && advancedEditors.length}
            <button type="button" class="filter-chip advanced-toggle" aria-expanded={moreExpanded} aria-controls={`${id}-advanced`}
                onclick={() => moreExpanded = !moreExpanded}>
                <i class="fa fa-ellipsis" aria-hidden="true"></i>
                <span>{$_(moreExpanded ? "trail-search-less" : "trail-search-more")}{advancedCount ? ` · ${advancedCount}` : ""}</span>
                <i class={`fa fa-chevron-${moreExpanded ? "up" : "down"}`} aria-hidden="true"></i>
            </button>
        {/if}
    </div>
    {@const currentEditor = editors.find(editor => editor.id === active)}
    {#if currentEditor}
        <section id={editorId} class="editor" aria-label={$_(currentEditor.label)}>
            {@render filterEditor(currentEditor.id)}
        </section>
    {/if}
{/snippet}

{#snippet directSuggestions()}
    <TrailSearchShortcuts bind:this={shortcuts} id={`${id}-shortcuts`} filter={draft}
        caret={queryCaret} selectionEnd={querySelectionEnd}
        onselect={captureShortcut} onremove={removeShortcut} onfocusinput={focusInput} />
    <TrailSearchSuggestions bind:this={suggestions} id={`${id}-suggestions`} query={draft.q}
        caret={queryCaret} selectionEnd={querySelectionEnd}
        onlocationselect={selectSuggestedLocation} />
{/snippet}

{#snippet sidebarGroup(editors: typeof availableEditors)}
    {#each editors as editor}
        {@const icon = sidebarHeadingIcon(editor.id, editor.icon)}
        <section class="sidebar-section" aria-labelledby={`${id}-sidebar-${editor.id}`}>
            <h3 id={`${id}-sidebar-${editor.id}`}>
                {#if icon}<i class={`fa ${icon}`} aria-hidden="true"></i>{/if}
                {$_(editor.label)}
            </h3>
            {@render filterEditor(editor.id)}
        </section>
    {/each}
{/snippet}

<svelte:window onpointerdown={handleOutsidePointer} onkeydown={handleKeydown} onresize={updatePanelHeight} onscroll={updatePanelHeight} />

<div bind:this={searchRoot} class="trail-search-bar" data-search-root class:with-inline-controls={inlineControls} class:in-flow={layout !== "overlay"} class:sidebar={pinned} onfocusout={handleFocusOut}>
    <form onsubmit={applySearch} class="search-form" class:expanded={expanded && !pinned} role="search" aria-label={$_("search-trails")}>
        <div bind:this={searchHeading} class="search-heading">
            <i class="fa fa-magnifying-glass" aria-hidden="true"></i>
            <input bind:this={searchInput} value={expanded ? draft.q : filter.q} type="search" autocomplete="off" aria-label={$_("search-trails")} placeholder={$_("trail-search-placeholder")}
                onfocus={() => { if (!restoringFocus) openSearch(); }}
                onclick={(event) => { openSearch(); updateQuerySelection(event); }}
                onselect={updateQuerySelection} onkeyup={updateQuerySelection} oninput={updateQuery}
                oncompositionstart={() => composing = true}
                oncompositionend={(event) => { composing = false; updateQuery(event); }}
                aria-controls={expanded ? `${id}-panel ${id}-shortcuts ${id}-suggestions` : `${id}-panel`} />
            <button type="button" class="icon-button reset-button" disabled={!resetEnabled}
                aria-label={$_("trail-search-reset")} title={$_("trail-search-reset")} onclick={resetSearch}>
                <i class="fa fa-arrow-rotate-left" aria-hidden="true"></i>
            </button>
            {#if !pinned}
                <button type="button" class="icon-button" aria-label={$_(expanded ? "trail-search-close" : "trail-search-filter-by")} aria-expanded={expanded} aria-controls={`${id}-panel`}
                    onclick={() => { if (expanded) dismiss(true); else { openSearch(); focusInput(); } }}>
                    <i class={`fa ${expanded ? "fa-xmark" : "fa-sliders"}`} aria-hidden="true"></i>
                </button>
            {/if}
        </div>
        {#if pinned}
            <div class="sidebar-suggestions">{@render directSuggestions()}</div>
        {/if}
        {#if expanded}
        <div bind:this={searchPanel} use:placeFilterPanel={pinned ? filterPanelTarget : undefined}
            id={`${id}-panel`} class="search-panel" class:sidebar-panel={pinned} role="region"
            aria-label={$_("trail-search-filter-by")} style:max-height={pinned ? undefined : `${panelMaxHeight}px`} onfocusout={handleFocusOut}>
            <div class="filter-body">
                {#if pinned}
                    <div class="general-filters sidebar-fields">{@render sidebarGroup(generalEditors)}</div>
                    {#if advancedEditors.length}
                        <div class="sidebar-more">
                            <button type="button" class="sidebar-more-toggle" aria-expanded={moreExpanded} aria-controls={`${id}-advanced`}
                                onclick={() => moreExpanded = !moreExpanded}>
                                <span>
                                    {$_("trail-search-more")}
                                    {#if advancedCount}<span class="advanced-count">{advancedCount}</span>{/if}
                                </span>
                                <i class={`fa fa-chevron-${moreExpanded ? "up" : "down"}`} aria-hidden="true"></i>
                            </button>
                            {#if moreExpanded}
                                <div id={`${id}-advanced`} class="advanced-filters sidebar-fields">{@render sidebarGroup(advancedEditors)}</div>
                            {/if}
                        </div>
                    {/if}
                {:else}
                    {@render directSuggestions()}
                    {#if generalEditors.length || advancedEditors.length}
                        <div class="general-filters">
                            {@render filterGroup(generalEditors, activeEditor, editor => activeEditor = editor, `${id}-editor`, $_("trail-search-filter-by"), true)}
                        </div>
                    {/if}
                    {#if moreExpanded && advancedEditors.length}
                        <div id={`${id}-advanced`} class="advanced-filters">
                            {@render filterGroup(advancedEditors, activeAdvancedEditor, editor => activeAdvancedEditor = editor, `${id}-advanced-editor`, $_("trail-search-more"))}
                        </div>
                    {/if}
                {/if}
            </div>

        </div>
        {/if}
    </form>
    {#if appliedChips.length}<SearchFilterSummary chips={appliedChips} onremove={removeApplied} />{/if}
</div>

<style>
    .trail-search-bar { position: relative; width: 100%; }
    button { cursor: pointer; }
    button, input { font: inherit; }
    button:focus-visible, input:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 3px; }
    .search-form { background: rgba(var(--input-background)); border-radius: .5rem; position: relative; }
    .with-inline-controls .search-heading { padding-right: calc(var(--trail-controls-width, 5.5rem) + var(--trail-controls-inset, .6rem) + .5rem); }
    .search-form:hover .search-heading, .search-form:focus-within .search-heading, .search-form.expanded .search-heading { border-color: rgba(var(--input-border-focus)); }
    .search-form.expanded { background: rgba(var(--menu-background)); border-radius: .5rem .5rem 0 0; z-index: 30; }
    .search-form.expanded .search-heading { border-bottom-color: transparent; }
    .search-panel { background: rgba(var(--menu-background)); border: 1px solid rgba(var(--input-border-focus)); border-top: 0; border-radius: 0 0 .5rem .5rem; box-shadow: 0 14px 24px rgb(0 0 0 / .12); color: rgba(var(--content)); container-type: inline-size; display: flex; flex-direction: column; left: 0; overflow: hidden; position: absolute; right: 0; top: calc(100% - 1px); }
    .search-heading { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: inherit; display: flex; gap: 1rem; min-height: 3.125rem; padding: .25rem 1.2rem; }
    .search-heading > i { color: rgba(var(--content), .65); font-size: 1.15rem; }
    .search-heading input { background: transparent; border: 0; flex: 1; font-size: 1rem; min-width: 0; outline: 0; padding: .35rem 0; text-overflow: ellipsis; }
    .search-heading input::placeholder { color: rgba(var(--content), .48); }
    .search-heading input:focus-visible { outline: none; }
    .icon-button { align-items: center; border-radius: 50%; display: flex; flex-shrink: 0; height: 2rem; justify-content: center; width: 2rem; }
    .icon-button:hover { background: rgba(var(--secondary-hover)); }
    .icon-button:disabled { cursor: default; opacity: .3; }
    .icon-button:disabled:hover { background: transparent; }
    .in-flow .search-panel { box-shadow: none; margin-bottom: -1px; position: relative; top: -1px; }
    .sidebar-panel { border: 1px solid rgba(var(--input-border)); border-radius: .5rem; box-shadow: none; margin: 0; overflow: visible; position: relative; top: 0; }
    .sidebar-panel .filter-body { overflow: visible; padding: 1rem; }
    .sidebar-panel .editor-help { display: none; }
    .sidebar-panel .activity-grid, .sidebar-panel .refinement .choice-row { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .sidebar-panel .activity-option { padding: .5rem .65rem; }
    .sidebar-panel .filter-chip { font-size: .8rem; gap: .45rem; padding-inline: .7rem; }
    .sidebar-panel .subcategory-choice { padding: .5rem .65rem; }
    .sidebar-fields { display: grid; gap: 1rem; }
    .sidebar-section { min-width: 0; }
    .sidebar-section + .sidebar-section { border-top: 1px solid rgba(var(--separator)); padding-top: 1rem; }
    .sidebar-section h3 { align-items: center; display: flex; font-size: .85rem; font-weight: 600; gap: .55rem; margin: 0 0 .75rem; }
    .sidebar-section h3 i { opacity: .65; }
    .sidebar-more { border-top: 1px solid rgba(var(--separator)); margin-top: 1rem; }
    .sidebar-more-toggle { align-items: center; display: flex; font-size: .85rem; font-weight: 600; gap: .75rem; justify-content: space-between; padding-block: .75rem; text-align: left; width: 100%; }
    .sidebar-more-toggle > span { align-items: center; display: flex; gap: .5rem; min-width: 0; }
    .sidebar-more-toggle > i { flex-shrink: 0; font-size: .65rem; }
    .advanced-count { align-items: center; background: rgba(var(--toggle-active)); border-radius: 50%; color: white; display: inline-flex; font-size: .65rem; height: 1.25rem; justify-content: center; min-width: 1.25rem; padding-inline: .25rem; }
    .sidebar-more .advanced-filters { border: 0; margin: 0; padding: .25rem 0 1rem; }
    .sidebar .search-form:focus-within:has(:global(.direct-suggestions), :global(.shortcut-suggestions), :global(.captured-filters)) .search-heading { border-bottom-color: transparent; border-radius: .5rem .5rem 0 0; }
    .sidebar-suggestions { background: rgba(var(--menu-background)); border: 1px solid rgba(var(--input-border-focus)); border-top: 0; border-radius: 0 0 .5rem .5rem; box-shadow: 0 8px 24px rgb(0 0 0 / .12); display: none; left: 0; max-height: min(24rem, 50vh); overflow-y: auto; padding: 1rem 1.2rem; position: absolute; right: 0; top: calc(100% - 1px); z-index: 30; }
    .search-form:focus-within .sidebar-suggestions:has(:global(.direct-suggestions), :global(.shortcut-suggestions), :global(.captured-filters)) { display: block; }
    .sidebar-suggestions :global(.direct-suggestions), .sidebar-suggestions :global(.shortcut-suggestions) { border-bottom: 0; margin-bottom: 0; padding-bottom: 0; }
    .filter-body { container-type: inline-size; min-height: 0; min-width: 0; overflow-y: auto; padding: 1.3rem 1.5rem 1.6rem; }
    .filter-caption { color: rgba(var(--content), .65); font-size: .8rem; font-weight: 600; margin: 0 0 .8rem; }
    .filter-chips { display: flex; flex-wrap: wrap; gap: .55rem; }
    .filter-chip { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: .5rem; display: inline-flex; font-size: .85rem; font-weight: 500; gap: .55rem; max-width: 100%; min-height: 2.55rem; min-width: 0; overflow-wrap: anywhere; padding: .55rem .95rem; transition: background .15s, border-color .15s; }
    .filter-chip > i { flex-shrink: 0; }
    .filter-chip > span { min-width: 0; }
    .filter-chip > i:first-child { font-size: .8rem; opacity: .7; }
    .filter-chip:hover, .choice:hover { background: rgba(var(--secondary-hover)); }
    .filter-chip.editing { border-color: rgba(var(--input-border-focus)); box-shadow: inset 0 0 0 1px rgba(var(--input-border-focus)); }
    .filter-chip.selected { background: rgba(var(--input-border-focus), .1); }
    .chip-check { font-size: .65rem; }
    .editor { padding-top: 1.1rem; }
    .editor-help { color: rgba(var(--content), .65); font-size: .85rem; line-height: 1.6; margin: 0 0 1.15rem; }
    .activity-grid { display: flex; flex-wrap: wrap; gap: .65rem; }
    .activity-option { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: .5rem; display: inline-flex; gap: .6rem; max-width: 100%; min-height: 2.75rem; min-width: 0; padding: .6rem .9rem; position: relative; text-align: left; }
    .activity-option:hover { background: rgba(var(--secondary-hover)); }
    .activity-option > span:first-of-type { flex: 1; font-size: .85rem; }
    .activity-option > span:first-of-type, .subcategory-choice > span:not(.subcategory-icon) { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .activity-icon { flex-shrink: 0; font-size: 1.1rem; opacity: .75; text-align: center; width: 1.35rem; }
    .chosen { background: rgba(var(--input-border-focus), .08); border-color: rgba(var(--input-border-focus)) !important; }
    .selection-check { background: rgba(var(--menu-background)); border: 1px solid rgba(var(--input-border-focus)); border-radius: 50%; font-size: .5rem; padding: .2rem; position: absolute; right: -.25rem; top: -.25rem; }
    .refinement { border-top: 1px solid rgba(var(--separator), .65); margin-top: 1.3rem; padding-top: 1rem; }
    .refinement-heading { align-items: center; display: flex; flex-wrap: wrap; font-size: .75rem; gap: .75rem; margin-bottom: .8rem; }
    .refinement-heading > span { color: rgba(var(--content), .55); }
    .category-switch { display: flex; flex-wrap: wrap; gap: .5rem; }
    .category-switch button { border-bottom: 2px solid transparent; font-size: .8rem; max-width: 100%; overflow: hidden; padding: .25rem .1rem; text-overflow: ellipsis; white-space: nowrap; }
    .category-switch button.current { border-color: rgba(var(--input-border-focus)); font-weight: 600; }
    .choice-row { display: flex; flex-wrap: wrap; gap: .65rem; }
    .choice { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: .5rem; display: inline-flex; font-size: .85rem; gap: .6rem; max-width: 100%; min-height: 2.6rem; min-width: 0; overflow-wrap: anywhere; padding: .6rem 1rem; }
    .subcategory-choice { position: relative; }
    .subcategory-icon { flex-shrink: 0; font-size: 1.1rem; line-height: 1.35rem; opacity: .75; position: relative; text-align: center; width: 1.5rem; }
    .subcategory-badge { background: rgba(var(--menu-background)); border-radius: .25rem; font-size: .5rem; padding: .1rem; position: absolute; right: -.15rem; top: -.1rem; }
    .advanced-filters { border-top: 1px solid rgba(var(--separator)); margin-top: 1.4rem; min-width: 0; padding-top: 1.25rem; }
    .advanced-toggle { align-items: center; display: inline-flex; font-size: .8rem; font-weight: 500; gap: .6rem; min-height: 2.5rem; min-width: 0; }
    .advanced-toggle > i { flex-shrink: 0; }
    .advanced-toggle > span { min-width: 0; overflow-wrap: anywhere; }
    .advanced-toggle > i:last-child { font-size: .6rem; margin-left: .15rem; }
    @media (max-width: 767px) {
        .search-form { border-radius: .5rem; }
        .search-form.expanded { border-radius: .5rem .5rem 0 0; }
        .search-panel { border-radius: 0 0 .5rem .5rem; }
        .search-heading { gap: .75rem; padding-inline: 1rem; }
        .search-heading input { font-size: 1rem; }
        .filter-body { padding: 1rem; }
        .filter-chips { gap: .45rem; }
        .filter-chip { font-size: .75rem; gap: .4rem; min-height: 2.5rem; padding: .5rem .75rem; }
    }

    @media (prefers-reduced-motion: reduce) { .search-form, .filter-chip { transition: none; } }
    @media (max-width: 639px) {
        .with-inline-controls .search-heading { gap: .5rem; padding-left: .65rem; }
        .with-inline-controls .search-heading > i { display: none; }
    }
</style>
