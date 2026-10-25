<script lang="ts">
    import {
        closeToast,
        pauseToast,
        resumeToast,
        toastStore,
        type Toast,
    } from "#lib/stores/toast_store.svelte.ts";
    import { _ } from "svelte-i18n";
    import { fade } from "svelte/transition";
</script>

{#snippet toast(t: Toast, i: number)}
    <!-- Hover and focus only pause the auto-close timer; the toast itself is not a control. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
        class="fixed px-4 py-3 shadow-xl rounded-xl border border-input-border right-4 bg-white text-gray-500 flex items-center max-w-md"
        style="z-index: 1001; bottom: {16 + i * 72}px"
        in:fade|global={{ duration: 250 }}
        out:fade|global={{ duration: 250 }}
        onmouseenter={() => pauseToast(t)}
        onmouseleave={() => resumeToast(t)}
        onfocusin={() => pauseToast(t)}
        onfocusout={() => resumeToast(t)}
    >
        <i
            class="fa fa-{t.icon} p-3 mr-2 rounded-lg"
            class:success-toast={t.type == "success"}
            class:error-toast={t.type == "error"}
            class:warning-toast={t.type == "warning"}
            aria-hidden="true"
        ></i>
        <p class="mr-4">
            {t.text}
        </p>
        <button
            type="button"
            class="text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm w-8 h-8 ms-auto inline-flex justify-center items-center dark:hover:bg-gray-600 dark:hover:text-white ml-6"
            onclick={() => closeToast(t)}
        >
            <i class="fa fa-close" aria-hidden="true"></i>
            <span class="sr-only">{$_("close")}</span>
        </button>
    </div>
{/snippet}

<!-- Screen readers only announce what is added inside a live region that
     already exists, so both regions stay mounted while empty. -->
<div role="status" aria-live="polite">
    {#each toastStore.toasts as t, i (t.id)}
        {#if t.type != "error"}
            {@render toast(t, i)}
        {/if}
    {/each}
</div>
<div role="alert" aria-live="assertive">
    {#each toastStore.toasts as t, i (t.id)}
        {#if t.type == "error"}
            {@render toast(t, i)}
        {/if}
    {/each}
</div>

<style>
    .success-toast {
        color: #10b981;
        background-color: #a7f3d0;
    }
    .error-toast {
        color: #ef4444;
        background-color: #fecaca;
    }
    .warning-toast {
        color: #ffb900;
        background-color: #fef3c6;
    }
</style>
