<script lang="ts">
    import emptyStateCommentDark from "#lib/assets/svgs/empty_states/empty_state_comment_dark.svg";
    import emptyStateCommentLight from "#lib/assets/svgs/empty_states/empty_state_comment_light.svg";
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
        src={$theme === "light" ? emptyStateCommentLight : emptyStateCommentDark}
        alt="Empty State showing a wanderer going into the distance"
    />

    <h3 class="text-xl font-medium text-center text-gray-500">{$_("no-comments-so-far")}...</h3>
</div>
