<script lang="ts">
    import emptyStateFeedDark from "#lib/assets/svgs/empty_states/empty_state_feed_dark.svg";
    import emptyStateFeedLight from "#lib/assets/svgs/empty_states/empty_state_feed_light.svg";
    import { theme } from "#lib/stores/theme_store.ts";

    import { _ } from "svelte-i18n";
</script>

<div class="flex flex-col justify-center items-center h-full">
    <img
        class="aspect-square h-96 mb-4"
        src={$theme === "light" ? emptyStateFeedLight : emptyStateFeedDark}
        alt="Empty State showing a wanderer going into the distance"
    />

    <h3 class="text-2xl font-medium text-center text-gray-500">{$_("empty-feed")}...</h3>
    <p class="text-center text-gray-500">
        {$_('empty-feed-explanation')}
    </p>
</div>
