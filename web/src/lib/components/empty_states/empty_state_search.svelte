<script lang="ts">
    import emptyStateSearchDark from "#lib/assets/svgs/empty_states/empty_state_search_dark.svg";
    import emptyStateSearchLight from "#lib/assets/svgs/empty_states/empty_state_search_light.svg";
    import { theme } from "#lib/stores/theme_store.ts";

    import { _ } from "svelte-i18n";
    interface Props {
        width?: number;
    }

    let { width = 256 }: Props = $props();
</script>

<div class="flex flex-col justify-center items-center gap-y-4">
    <img
        style="width: {width}px"
        class="rounded-full aspect-square"
        src={$theme === "light" ? emptyStateSearchLight : emptyStateSearchDark}
        alt="Empty State showing a wanderer going into the distance"
    />

    <h3 class="text-2xl font-semibold text-center text-gray-500">{$_("no-results")}</h3>
</div>
