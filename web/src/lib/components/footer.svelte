<script>
    import LogoTextLightWithVersion from "./logo/logo_text_light_with_version.svelte";
    import { _ } from "svelte-i18n";
</script>

<footer
    class="bg-footer-background text-white w-full grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-y-8 gap-x-12 p-12 mt-12 rounded-t-3xl"
>
    <div class="col-span-2 md:col-span-1">
        <LogoTextLightWithVersion></LogoTextLightWithVersion>
    </div>
    <div>
        <h5 class="font-semibold">Resources</h5>
        <ul class="mt-4 text-sm">
            <li>
                <a href="https://wanderer.to/run/installation/"
                    >{$_("documentation")}</a
                >
            </li>
            <li>
                <a href="https://wanderer.to/api-reference"
                    >{$_("api-documentation")}</a
                >
            </li>
            <li><a href="https://pocketbase.io/">PocketBase</a></li>
            <li><a href="https://www.meilisearch.com/">meilisearch</a></li>
        </ul>
    </div>
    <div>
        <h5 class="font-semibold">wanderer</h5>
        <ul class="mt-4 text-sm">
            <li>
                <a href="https://wanderer.to">{$_("about")}</a>
            </li>
            <li>
                <a href="https://wanderer.to/#features">{$_("features")}</a>
            </li>
            <li>
                <a href="https://wanderer.to/changelog/"
                    >{$_("changelog")}</a
                >
            </li>
            <li>
                <a href="https://github.com/open-wanderer/wanderer/blob/main/LICENSE"
                    >{$_("license")}</a
                >
            </li>
        </ul>
    </div>
    <div>
        <h5 class="font-semibold">Community</h5>
        <ul class="mt-4 text-sm">
            <li><a href="https://github.com/open-wanderer/wanderer">GitHub</a></li>
            <li>
                <a href="https://github.com/open-wanderer/wanderer/issues">Issues</a>
            </li>
            <li>
                <a href="https://github.com/open-wanderer/wanderer/pulls"
                    >{$_("contribute")}</a
                >
            </li>
        </ul>
    </div>
</footer>

<style>
    footer li:not(:last-child) {
        margin-bottom: 4px;
    }
</style>
