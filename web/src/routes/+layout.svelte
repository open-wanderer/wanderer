<script lang="ts">
    import {
        beforeNavigate,
        goto,
        invalidate,
        invalidateAll,
    } from "$app/navigation";

    import { page } from "$app/state";
    import { env } from "$env/dynamic/public";
    import Toast from "$lib/components/base/toast.svelte";
    import Footer from "$lib/components/footer.svelte";
    import NavBar from "$lib/components/nav_bar.svelte";
    import PageLoadingBar from "$lib/components/page_loading_bar.svelte";
    import UploadDialog from "$lib/components/settings/upload_dialog.svelte";
    import { currentUser } from "$lib/stores/user_store";
    import { isRouteProtected } from "$lib/util/authorization_util";
    import { onMount, type Snippet } from "svelte";
    import { slide } from "svelte/transition";
    import "../css/app.css";
    import "../css/components.css";
    import "../css/theme.css";
    import type { LayoutData } from "./$types";
    import PocketBase from "pocketbase";
    import { browser } from "$app/environment";

    interface Props {
        data: LayoutData;
        children?: Snippet;
    }

    let { data, children }: Props = $props();

    beforeNavigate((n) => {
        if (!$currentUser && isRouteProtected(n.to?.url)) {
            n.cancel();
            goto("/login?r=" + n.to?.url?.pathname);
        }
    });

    onMount(() => {
        if (page.data.origin != location.origin) {
            showWarning = true;
        }
    });

    let hideDemoHint = $state(false);
    let showWarning = $state(false);
</script>

{#if env.PUBLIC_IS_DEMO === "true" && !hideDemoHint}
    <div
        class="flex items-center justify-between bg-amber-200 text-center p-4 text-sm text-black"
        out:slide
    >
        <div></div>
        <span
            >This is a demo instance. Do not store any relevant data here. You
            can use the user 'demo' and password 'password' to login.
        </span>
        <button
            aria-label="Close"
            class="btn-icon self-end"
            onclick={() => (hideDemoHint = true)}
            ><i class="fa fa-close"></i></button
        >
    </div>
{/if}

{#if showWarning}
    <div
        class="flex items-center justify-between bg-red-200 text-center p-4 text-sm text-black"
        out:slide
    >
        <div></div>
        <p>
            You are accessing wanderer from <span class="font-mono bg-gray-100"
                >{location.origin}</span
            >
            but your ORIGIN environment variable is set to
            <span class="font-mono bg-gray-100">{page.data.origin}</span>. This
            may cause errors.
        </p>
        <button
            aria-label="Close"
            class="btn-icon self-end"
            onclick={() => (showWarning = false)}
            ><i class="fa fa-close"></i></button
        >
    </div>
{/if}

<NavBar user={data.user} mdPages={data.mdPages}></NavBar>
<PageLoadingBar class="text-content"></PageLoadingBar>
<Toast></Toast>
<UploadDialog></UploadDialog>
{@render children?.()}

<Footer></Footer>
