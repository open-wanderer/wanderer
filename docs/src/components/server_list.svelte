<script lang="ts">
    import type { Server } from "../models/server";
    import type { SelectItem } from "../models/select_item";

    interface Props {
        servers: Server[];
    }
    let { servers }: Props = $props();

    let search = $state("");
    let region = $state("");
    let language = $state("");
    let category = $state("");

    function uniqueValues(key: "region" | "language" | "category") {
        const allValues = servers.flatMap((s) => s[key]);
        return [...new Set(allValues)].sort();
    }

    const regionItems: SelectItem[] = [
        { text: "All regions", value: "" },
        ...uniqueValues("region").map((v) => ({ text: v, value: v })),
    ];

    const languageItems: SelectItem[] = [
        { text: "All languages", value: "" },
        ...uniqueValues("language").map((v) => ({ text: v, value: v })),
    ];

    const categoryItems: SelectItem[] = [
        { text: "All categories", value: "" },
        ...uniqueValues("category").map((v) => ({ text: v, value: v })),
    ];

    const filteredServers = $derived(
        servers.filter(
            (server) =>
                (server.name.toLowerCase().includes(search.toLowerCase()) ||
                    server.description
                        .toLowerCase()
                        .includes(search.toLowerCase())) &&
                (!region || server.region.includes(region)) &&
                (!language || server.language.includes(language)) &&
                (!category || server.category.includes(category)),
        ),
    );
</script>

<!-- ─── FILTERS ───────────────────────────────────────────────────────────── -->
<div class="filters" role="search" aria-label="Filter servers">
    <div class="filter-search">
        <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        <input
            class="search-input"
            type="search"
            placeholder="Search servers…"
            bind:value={search}
            aria-label="Search servers"
        />
    </div>

    <div class="filter-selects">
        <select class="filter-select" bind:value={region} aria-label="Filter by region">
            {#each regionItems as item}
                <option value={item.value}>{item.text}</option>
            {/each}
        </select>

        <select class="filter-select" bind:value={language} aria-label="Filter by language">
            {#each languageItems as item}
                <option value={item.value}>{item.text}</option>
            {/each}
        </select>

        <select class="filter-select" bind:value={category} aria-label="Filter by category">
            {#each categoryItems as item}
                <option value={item.value}>{item.text}</option>
            {/each}
        </select>
    </div>

    {#if search || region || language || category}
        <button
            class="filter-clear"
            onclick={() => { search = ""; region = ""; language = ""; category = ""; }}
            aria-label="Clear all filters"
        >
            Clear
        </button>
    {/if}
</div>

<!-- ─── RESULTS META ──────────────────────────────────────────────────────── -->
<div class="results-meta" aria-live="polite" aria-atomic="true">
    {filteredServers.length}
    {filteredServers.length === 1 ? "server" : "servers"}
    {search || region || language || category ? "found" : "listed"}
</div>

<!-- ─── SERVER GRID ───────────────────────────────────────────────────────── -->
{#if filteredServers.length === 0}
    <div class="empty-state" role="status">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        <p class="empty-heading">No servers match your filters.</p>
        <p class="empty-sub">Try broadening your search or clearing the filters.</p>
    </div>
{:else}
    <ul class="server-grid" aria-label="Server list">
        {#each filteredServers as server}
            <li class="server-card">
                <div class="card-image-wrap">
                    <img
                        src={server.image}
                        alt=""
                        class="card-image"
                        loading="lazy"
                    />
                </div>

                <div class="card-tags" aria-label="Server tags">
                    {#each [...server.region, ...server.language, ...server.category] as tag}
                        <span class="tag">{tag}</span>
                    {/each}
                </div>

                <div class="card-body">
                    <h2 class="card-name">{server.name}</h2>
                    <p class="card-description">{server.description}</p>
                </div>

                <div class="card-footer">
                    <a
                        href={server.url}
                        class="card-cta"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        Create account
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                    </a>
                </div>
            </li>
        {/each}
    </ul>
{/if}

<style>
    /* ── Design tokens (mirrors servers.astro :root) ── */
    :global(:root) {
        --deep-void: #12141c;
        --surface-dark: #1e2028;
        --carbon: #262831;
        --dusk-divide: #374151;
        --summit-fog: #94a3b8;
        --polar-night: #242734;
        --polar-night-lifted: #373c50;
        --white: #ffffff;
    }

    /* ── Filters ──────────────────────────────────────────────────────────── */
    .filters {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        flex-wrap: wrap;
        padding: 1.25rem 0;
        border-bottom: 1px solid var(--dusk-divide);
        margin-bottom: 1.25rem;
    }

    .filter-search {
        position: relative;
        flex: 1;
        min-width: 200px;
    }

    .search-icon {
        position: absolute;
        left: 0.875rem;
        top: 50%;
        transform: translateY(-50%);
        color: var(--summit-fog);
        pointer-events: none;
    }

    .search-input {
        width: 100%;
        background: var(--surface-dark);
        border: 1px solid var(--dusk-divide);
        border-radius: 6px;
        padding: 0.625rem 0.875rem 0.625rem 2.5rem;
        font-family: inherit;
        font-size: 0.9375rem;
        color: var(--white);
        outline: none;
        transition: border-color 0.15s ease;
        appearance: none;
        -webkit-appearance: none;
    }
    .search-input::placeholder {
        color: var(--summit-fog);
    }
    .search-input:focus {
        border-color: var(--summit-fog);
    }
    /* hide browser search clear button */
    .search-input::-webkit-search-cancel-button { display: none; }

    .filter-selects {
        display: flex;
        gap: 0.75rem;
        flex-wrap: wrap;
    }

    .filter-select {
        background: var(--surface-dark);
        border: 1px solid var(--dusk-divide);
        border-radius: 6px;
        padding: 0.625rem 2rem 0.625rem 0.875rem;
        font-family: inherit;
        font-size: 0.9375rem;
        color: var(--white);
        outline: none;
        cursor: pointer;
        transition: border-color 0.15s ease;
        appearance: none;
        -webkit-appearance: none;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%2394a3b8' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 0.625rem center;
    }
    .filter-select:focus {
        border-color: var(--summit-fog);
    }
    .filter-select option {
        background: var(--carbon);
        color: var(--white);
    }

    .filter-clear {
        padding: 0.625rem 0.875rem;
        font-family: inherit;
        font-size: 0.875rem;
        font-weight: 500;
        color: var(--summit-fog);
        background: transparent;
        border: 1px solid var(--dusk-divide);
        border-radius: 6px;
        cursor: pointer;
        transition: color 0.15s ease, border-color 0.15s ease;
        flex-shrink: 0;
    }
    .filter-clear:hover {
        color: var(--white);
        border-color: rgba(255, 255, 255, 0.3);
    }

    /* ── Results meta ─────────────────────────────────────────────────────── */
    .results-meta {
        font-size: 0.8125rem;
        color: var(--summit-fog);
        margin-bottom: 1.5rem;
    }

    /* ── Server grid ──────────────────────────────────────────────────────── */
    .server-grid {
        list-style: none;
        padding: 0;
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 1.25rem;
    }

    /* ── Server card ──────────────────────────────────────────────────────── */
    .server-card {
        display: flex;
        flex-direction: column;
        background: var(--surface-dark);
        border: 1px solid var(--dusk-divide);
        border-radius: 8px;
        overflow: hidden;
        transition: border-color 0.2s ease;
    }
    .server-card:hover {
        border-color: rgba(255, 255, 255, 0.2);
    }

    .card-image-wrap {
        height: 160px;
        overflow: hidden;
        background: var(--carbon);
        flex-shrink: 0;
    }

    .card-image {
        width: 100%;
        height: 100%;
        object-fit: cover;
        display: block;
        transition: transform 0.35s cubic-bezier(0.33, 1, 0.68, 1);
    }
    .server-card:hover .card-image {
        transform: scale(1.03);
    }

    .card-tags {
        display: flex;
        flex-wrap: wrap;
        gap: 0.375rem;
        padding: 0.875rem 1rem 0;
    }

    .tag {
        display: inline-block;
        padding: 0.1875rem 0.5rem;
        font-size: 0.75rem;
        font-weight: 500;
        color: var(--summit-fog);
        border: 1px solid var(--dusk-divide);
        border-radius: 4px;
        line-height: 1.5;
    }

    .card-body {
        padding: 0.75rem 1rem 0;
        flex: 1;
    }

    .card-name {
        font-size: 1rem;
        font-weight: 600;
        line-height: 1.35;
        color: var(--white);
        margin-bottom: 0.4rem;
    }

    .card-description {
        font-size: 0.875rem;
        line-height: 1.6;
        color: var(--summit-fog);
    }

    .card-footer {
        padding: 1rem;
        margin-top: 0.5rem;
        display: flex;
    }

    .card-cta {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        flex: 1;
        padding: 0.625rem 1rem;
        font-family: inherit;
        font-size: 0.9375rem;
        font-weight: 600;
        color: var(--white);
        background: var(--polar-night);
        border: 1px solid transparent;
        border-radius: 6px;
        text-decoration: none;
        cursor: pointer;
        transition: background 0.15s ease;
    }
    .card-cta:hover {
        background: var(--polar-night-lifted);
    }
    .card-cta:focus-visible {
        outline: 2px solid var(--summit-fog);
        outline-offset: 3px;
    }

    /* ── Empty state ──────────────────────────────────────────────────────── */
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 0.75rem;
        padding: 5rem 1rem;
        color: var(--summit-fog);
        text-align: center;
    }

    .empty-heading {
        font-size: 1.125rem;
        font-weight: 600;
        color: var(--white);
        margin: 0;
    }

    .empty-sub {
        font-size: 0.9375rem;
        color: var(--summit-fog);
        margin: 0;
    }

    /* ── Responsive ───────────────────────────────────────────────────────── */
    @media (max-width: 600px) {
        .filter-search {
            min-width: 100%;
        }
        .filter-selects {
            width: 100%;
        }
        .filter-select {
            flex: 1;
            min-width: 0;
        }
    }
</style>
