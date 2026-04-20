<script>
    import { page } from "$app/state";
    import errorDark from "$lib/assets/svgs/empty_states/error_dark.svg";
    import errorLight from "$lib/assets/svgs/empty_states/error_light.svg";
    import { theme } from "$lib/stores/theme_store";
</script>

<svelte:head>
    <title>{page.status} | {page.data.appTitle}</title>
</svelte:head>
<div
    class="m-auto text-center max-w-md"
    style="height: calc(100dvh - 120px); font-size: 4rem"
>
    <img
        class="aspect-square rounded-xl mx-auto"
        width="300"
        src={$theme === "light" ? errorLight : errorDark}
        alt="Default error img"
    />
    <div class="text-center mt-8">
        <h1 class="font-bold" style="line-height: 4rem;">
            {page.status}
        </h1>
        <p class="text-lg text-center mt-2">
            {page.error?.message}
        </p>
    </div>
</div>
