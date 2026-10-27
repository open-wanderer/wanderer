<script lang="ts">
    import emptyStateDescriptionDark from "#lib/assets/svgs/empty_states/empty_state_description_dark.svg";
    import emptyStateDescriptionLight from "#lib/assets/svgs/empty_states/empty_state_description_light.svg";
    import { theme } from "#lib/stores/theme_store.ts";

    import { _ } from "svelte-i18n";
    interface Props {
        width?: number;
    }

    let { width = 192 }: Props = $props();
</script>

<div class="flex flex-col justify-center items-center gap-y-4">
    <img
        style="width: {width}px"
        class="rounded-full aspect-square"
        src={$theme === "light" ? emptyStateDescriptionLight : emptyStateDescriptionDark}
        alt="Empty State showing a wanderer going into the distance"
    />

    <h3 class="text-xl font-medium text-center text-gray-500">{$_("no-description-for-now")}...</h3>
</div>
