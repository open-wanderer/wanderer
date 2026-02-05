<script lang="ts">
    import { page } from "$app/state";
    import TrailList from "$lib/components/trail/trail_list.svelte";
    import type { TrailFilter } from "$lib/models/trail.js";
    import { profile_trails_index } from "$lib/stores/profile_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    let loading = $state(true);

    let pagination = $state({
        page: data.trails.page,
        totalPages: data.trails.totalPages,
        items: 12,
    });

    let trails = $state(data.trails);

    let filter: TrailFilter = $state(data.filter);

    async function handleFilterUpdate() {
        loading = true;
        try {
            trails = await profile_trails_index(
                page.params.handle!,
                filter,
                pagination.page,
                pagination.items,
                fetch,
            );
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

    async function paginate(newPage: number, items?: number) {
        pagination.page = newPage;
        try {
            trails = await profile_trails_index(
                page.params.handle!,
                filter,
                newPage,
                items ?? pagination.items,
                fetch,
            );
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
    {pagination}
    {loading}
    fullWidthCards={true}
    trails={trails.items}
    {filter}
    onupdate={handleFilterUpdate}
    onpagination={paginate}
></TrailList>
