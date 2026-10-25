<script lang="ts">
    import { page } from "$app/state";
    import TrailList from "$lib/components/trail/trail_list.svelte";
    import type { TrailFilter } from "$lib/models/trail.js";
    import { profile_trails_index } from "$lib/stores/profile_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import { untrack } from "svelte";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    let loading = $state(true);

    let pagination = $state({
        page: untrack(() => data.trails.page),
        totalPages: untrack(() => data.trails.totalPages),
        items: 12,
    });

    let trails = $state(untrack(() => data.trails));

    let filter: TrailFilter = $state(untrack(() => data.filter));

    function handleFilterUpdate() {
        return paginate(pagination.page);
    }

    async function paginate(newPage: number, items: number = pagination.items) {
        loading = true;
        try {
            const response = await profile_trails_index(
                page.params.handle!,
                filter,
                newPage,
                items,
                fetch,
            );
            trails = response;
            pagination.page = response.page;
            pagination.totalPages = response.totalPages;
            pagination.items = items;
        } catch (e) {
            show_toast({
                icon: "close",
                text: "Error loading trails.",
                type: "error",
            });
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>{$_("profile")} | wanderer</title>
</svelte:head>
<TrailList
    bind:pagination
    {loading}
    fullWidthCards={true}
    bind:trails={trails.items}
    {filter}
    onupdate={handleFilterUpdate}
    onpagination={paginate}
></TrailList>
