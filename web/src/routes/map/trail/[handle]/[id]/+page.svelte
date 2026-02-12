<script lang="ts">
    import { page } from "$app/state";
    import MetaTags from "$lib/components/base/meta_tags.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import TrailInfoPanel from "$lib/components/trail/trail_info_panel.svelte";
    import { getFileURL } from "$lib/util/file_util.js";
    import * as M from "maplibre-gl";
    import "photoswipe/style.css";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    const trail = $state(data.trail);

    let markers: M.Marker[] = $state([]);
</script>


<MetaTags
    title={`${data.trail.name} | ${$_("map")} | wanderer`}
    openGraph={{
        title: data.trail.name,
        description: data.trail.description,
        url: page.url.origin + page.url.pathname,
        type: "article",
        article: {
            publishedTime: data.trail.created,
            modifiedTime: data.trail.updated,
            authors: [
                `${page.url.origin}/profile/@${data.trail.expand?.author?.preferred_username ?? ""}`,
            ],
            tags: data.trail.expand?.tags?.map((t) => t.name),
        },
        images: data.trail.photos.map((p) => ({
            url: page.url.origin + getFileURL(data.trail, p),
        })),
    }}
></MetaTags>

<main class="grid grid-cols-1 md:grid-cols-[458px_1fr] gap-x-1 gap-y-4">
    <div id="panel" class="hidden md:block">
        <TrailInfoPanel handle={page.params.handle!} initTrail={trail} {markers}
        ></TrailInfoPanel>
    </div>
    <div id="trail-details">
        <MapWithElevationMaplibre
            trails={[trail]}
            waypoints={trail.expand?.waypoints_via_trail}
            activeTrail={0}
            bind:markers
            showTerrain={true}
        ></MapWithElevationMaplibre>
    </div>
</main>

<style>
    #trail-details,
    #panel {
        height: calc(100vh);
    }
    @media only screen and (min-width: 768px) {
        #trail-details,
        #panel {
            height: calc(100vh - 124px);
        }
    }
</style>
