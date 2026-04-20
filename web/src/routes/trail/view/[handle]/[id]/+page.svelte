<script lang="ts">
    import { page } from "$app/state";
    import MetaTags from "$lib/components/base/meta_tags.svelte";
    import TrailInfoPanel from "$lib/components/trail/trail_info_panel.svelte";
    import { trail } from "$lib/stores/trail_store";
    import { getFileURL } from "$lib/util/file_util.js";
    import { _ } from "svelte-i18n";

    let { data } = $props();
</script>

{#if data.trail}
    <MetaTags
        title={`${data.trail.name} | ${$_("trail", { values: { n: 1 } })} | ${page.data.appTitle}`}
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
    <TrailInfoPanel
        activeTab={parseInt(page.url.searchParams.get("t") ?? "0")}
        initTrail={data.trail}
        mode="overview"
        handle={page.params.handle!}
    ></TrailInfoPanel>
{/if}
