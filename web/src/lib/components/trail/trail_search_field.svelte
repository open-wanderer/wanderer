<script lang="ts">
    import type { ActorSearchResult } from "$lib/models/activitypub/actor";
    import type { TrailSearchFilterId } from "$lib/models/trail_search_settings";
    import type { TrailSearchDraft } from "$lib/util/trail_search_draft";
    import { tags_index } from "$lib/stores/tag_store";
    import { currentUser } from "$lib/stores/user_store";
    import { formatElevation } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import ActorSearch from "../actor_search.svelte";
    import Combobox, { type ComboboxItem } from "../base/combobox.svelte";
    import MultiSelect from "../base/multi_select.svelte";
    import type { SelectItem } from "../base/select.svelte";
    import type { SearchItem } from "../base/search.svelte";
    import TrailSearchRange from "./trail_search_range.svelte";

    interface Props {
        filter: TrailSearchDraft;
        field: TrailSearchFilterId;
        authorLabel?: string;
        onauthorselect?: (author: ActorSearchResult) => void;
    }

    let { filter = $bindable(), field, authorLabel, onauthorselect }: Props = $props();
    const id = $props.id();

    let tagItems: ComboboxItem[] = $state([]);
    let tagRequest = 0;
    let authorQuery = $state("");

    const elevationRanges = [
        {
            key: "gain",
            label: "elevation-gain",
            icon: "arrow-trend-up",
            min: "elevationGainMin",
            max: "elevationGainMax",
            limit: "elevationGainLimit",
        },
        {
            key: "loss",
            label: "elevation-loss",
            icon: "arrow-trend-down",
            min: "elevationLossMin",
            max: "elevationLossMax",
            limit: "elevationLossLimit",
        },
    ] as const;

    const difficultyItems = $derived([
        { text: $_("easy"), value: 0 },
        { text: $_("moderate"), value: 1 },
        { text: $_("difficult"), value: 2 },
    ]);

    function getDifficulties(): SelectItem[] {
        return filter.difficulty.length === 3
            ? []
            : difficultyItems.filter(item => filter.difficulty.includes(item.value as 0 | 1 | 2));
    }

    function setDifficulties(items: SelectItem[]) {
        filter.difficulty = items.length ? items.map(item => item.value as 0 | 1 | 2) : [0, 1, 2];
    }

    // These existing search components do not expose an input ID prop.
    function identifyInput(node: HTMLDivElement, inputId: string) {
        const input = node.querySelector("input");
        if (input) {
            input.id = inputId;
            input.setAttribute("aria-labelledby", `${inputId}-label`);
        }
    }

    async function searchTags(query: string) {
        const request = ++tagRequest;
        const result = await tags_index(query);
        if (request === tagRequest) {
            tagItems = result.items.map((tag) => ({ text: tag.name, value: tag }));
        }
    }

    function getTags(): ComboboxItem[] {
        return filter.tags.map((tag) => ({ text: tag, value: tag }));
    }

    function setTags(tags: ComboboxItem[]) {
        filter.tags = tags.map((tag) => tag.text);
    }

    function selectAuthor(item: SearchItem) {
        filter.author = item.value.id;
        onauthorselect?.(item.value);
    }

    function clearAuthor() {
        filter.author = "";
        authorQuery = "";
    }
</script>

{#if field === "difficulty"}
    <div class="difficulty-editor">
        <MultiSelect name={`${id}-difficulty`} label={$_("difficulty")} items={difficultyItems}
            bind:value={getDifficulties, setDifficulties} placeholder={`${$_("filter-difficulty")}...`} />
    </div>
{:else if field === "tags"}
    <section class="filter-group" aria-labelledby={`${id}-tags-label`}>
        <h3 id={`${id}-tags-label`}>
            <i class="fa fa-tags" aria-hidden="true"></i>
            {$_("tags")}
        </h3>
        <div class="search-control" use:identifyInput={`${id}-tags`}>
            <Combobox
                name={`${id}-tags`}
                bind:value={getTags, setTags}
                onupdate={searchTags}
                placeholder={`${$_("filter-tags")}...`}
                items={tagItems}
                multiple
                chips
            />
        </div>
    </section>
{:else if field === "author" && $currentUser}
    <section class="filter-group" aria-labelledby={`${id}-author-label`}>
        <h3 id={`${id}-author-label`}>
            <i class="fa fa-user" aria-hidden="true"></i>
            {$_("author")}
        </h3>
        <div class="search-control author-control" use:identifyInput={`${id}-author`}>
            <ActorSearch bind:value={authorQuery} placeholder={$_("trail-search-author-placeholder")} onclick={selectAuthor} />
        </div>
        {#if filter.author}
            <button
                type="button"
                class="author-chip"
                aria-label={`${$_("remove")}: ${$_("author")}`}
                onclick={clearAuthor}
            >
                <span>{authorLabel ?? $_("author")}</span>
                <i class="fa fa-close" aria-hidden="true"></i>
            </button>
        {/if}
    </section>
{:else if field === "visibility" && $currentUser}
    <fieldset class="filter-group">
        <legend>{$_("visibilty-status")}</legend>
        <div class="choices">
            <label class="choice" for={`${id}-private`}>
                <input
                    id={`${id}-private`}
                    type="checkbox"
                    bind:checked={filter.private}
                />
                <span>{$_("private")}</span>
            </label>
            <label class="choice" for={`${id}-public`}>
                <input
                    id={`${id}-public`}
                    type="checkbox"
                    bind:checked={filter.public}
                />
                <span>{$_("public")}</span>
            </label>
            <label class="choice" for={`${id}-shared`}>
                <input
                    id={`${id}-shared`}
                    type="checkbox"
                    bind:checked={filter.shared}
                />
                <span>{$_("shared")}</span>
            </label>
        </div>
    </fieldset>
{:else if field === "completed"}
    <fieldset class="filter-group">
        <legend>{$_("completion-status")}</legend>
        <div class="choices">
            {#each [undefined, true, false] as completed, index}
                <label class="choice" for={`${id}-completed-${index}`}>
                    <input
                        id={`${id}-completed-${index}`}
                        type="radio"
                        name={`${id}-completed`}
                        checked={filter.completed === completed}
                        onchange={() => (filter.completed = completed)}
                    />
                    <span>{$_(completed === undefined
                        ? "no-preference"
                        : completed
                          ? "completed"
                          : "not-completed")}</span>
                </label>
            {/each}
        </div>
    </fieldset>
{:else if field === "gain" || field === "loss"}
    {@const range = elevationRanges.find(item => item.key === field)!}
    <TrailSearchRange label={$_(range.label)} icon={`fa-${range.icon}`}
        bind:min={filter[range.min]} bind:max={filter[range.max]} limit={filter[range.limit]}
        formatValue={formatElevation} />
{:else if field === "date"}
    <fieldset class="filter-group">
        <legend>{$_("date")}</legend>
        <div class="date-fields">
            <div>
                <label class="field-label" for={`${id}-after`}>{$_("after")}</label>
                <input
                    id={`${id}-after`}
                    type="date"
                    max={filter.endDate || undefined}
                    bind:value={filter.startDate}
                />
            </div>
            <div>
                <label class="field-label" for={`${id}-before`}>{$_("before")}</label>
                <input
                    id={`${id}-before`}
                    type="date"
                    min={filter.startDate || undefined}
                    bind:value={filter.endDate}
                />
            </div>
        </div>
    </fieldset>
{:else if field === "liked"}
    <fieldset class="filter-group">
        <legend>{$_("like-status")}</legend>
        <label class="choice" for={`${id}-liked`}>
            <input id={`${id}-liked`} type="checkbox" bind:checked={filter.liked} />
            <span>{$_("liked")}</span>
        </label>
    </fieldset>
{/if}

<style>
    .filter-group {
        border: 0;
        margin: 0;
        min-width: 0;
        padding: 0;
    }

    h3,
    .difficulty-editor :global(label),
    legend { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; }

    .search-control {
        max-width: 36rem;
        min-width: 0;
    }

    .search-control :global(input) {
        font-size: .85rem;
        min-width: 0;
    }

    .author-control :global(input) {
        min-height: 3.125rem;
    }

    .author-control :global(.fa-search) {
        color: rgba(var(--content), .6);
        font-size: 1rem;
        width: 1rem;
    }

    .choices {
        display: flex;
        flex-wrap: wrap;
        gap: .45rem 1.5rem;
    }

    .choice {
        align-items: center;
        cursor: pointer;
        display: inline-flex;
        font-size: .8rem;
        gap: .5rem;
        min-height: 2.5rem;
        padding: .5rem 0;
    }

    .author-chip {
        align-items: center;
        border: 1px solid rgba(var(--input-border));
        border-radius: .5rem;
        cursor: pointer;
        display: inline-flex;
        font-size: .8rem;
        gap: .5rem;
        min-height: 2.5rem;
        padding: .5rem .7rem;
    }

    .author-chip:hover {
        background: rgba(var(--secondary-hover));
    }

    .difficulty-editor {
        max-width: 36rem;
    }

    .difficulty-editor:has(:global([aria-expanded="true"])) {
        padding-bottom: 10.5rem;
    }

    input[type="checkbox"],
    input[type="radio"] {
        accent-color: rgba(var(--input-border-focus));
        flex-shrink: 0;
        height: 1rem;
        width: 1rem;
    }

    .author-chip {
        margin-top: .5rem;
        max-width: 100%;
    }

    .author-chip span {
        overflow-wrap: anywhere;
    }

    .date-fields {
        display: grid;
        gap: .8rem;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        max-width: 28rem;
    }

    .date-fields > div {
        min-width: 0;
    }

    .field-label {
        display: block;
        font-size: .75rem;
        margin-bottom: .35rem;
    }

    input[type="date"] {
        background: rgba(var(--input-background));
        border: 1px solid rgba(var(--input-border));
        border-radius: .5rem;
        font-size: .8rem;
        min-width: 0;
        padding: .65rem;
        width: 100%;
    }

    @container (max-width: 28rem) {
        .date-fields {
            grid-template-columns: minmax(0, 1fr);
        }
    }

    :global(.dark) input[type="date"] {
        color-scheme: dark;
    }

    input:focus-visible,
    .author-chip:focus-visible {
        outline: 2px solid rgba(var(--input-border-focus));
        outline-offset: 3px;
    }

</style>
