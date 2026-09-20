<script lang="ts">
    import {
        processUploadQueue,
        uploadStore,
        type Upload,
    } from "$lib/stores/upload_store.svelte";
    import { slide } from "svelte/transition";
    import { _ } from "svelte-i18n";
    import { goto } from "$app/navigation";
    import { currentUser } from "$lib/stores/user_store";

    let minimized: boolean = $state(false);

    let visibleUploads = $derived(
        uploadStore.enqueuedUploads
            .concat(uploadStore.completedUploads)
            .sort((a, b) => a.file.size - b.file.size),
    );

    let remaining = $derived(uploadStore.enqueuedUploads.length);

    let successfulUploads = $derived(
        uploadStore.completedUploads.reduce(
            (sum, u) => (sum += u.status == "success" ? 1 : 0),
            0,
        ),
    );

    let errorUploads = $derived(
        uploadStore.completedUploads.reduce(
            (sum, u) => (sum += u.status == "error" ? 1 : 0),
            0,
        ),
    );

    let duplicateUploads = $derived(
        uploadStore.completedUploads.reduce(
            (sum, u) => (sum += u.status == "duplicate" ? 1 : 0),
            0,
        ),
    );

    function dismissUpload(u: Upload) {
        const index = uploadStore.completedUploads.indexOf(u);
        uploadStore.completedUploads.splice(index, 1);
    }

    function dismissAllCompleted() {
        uploadStore.completedUploads = [];
    }

    function cancelUpload(u: Upload) {
        const index = uploadStore.enqueuedUploads.indexOf(u);
        u.status = "cancelled";
        uploadStore.enqueuedUploads.splice(index, 1);
        uploadStore.completedUploads.push(u);
    }

    function reUpload(u: Upload, ignoreDuplicates: boolean = false) {
        const index = uploadStore.completedUploads.indexOf(u);
        u.status = "enqueued";
        u.progress = 0;
        u.error = undefined;
        u.duplicate = undefined;
        uploadStore.completedUploads.splice(index, 1);
        uploadStore.enqueuedUploads.push(u);
        processUploadQueue(undefined, ignoreDuplicates);
    }
</script>

{#if visibleUploads.length}
    <div
        class="fixed bottom-4 right-4 z-10 p-4 bg-background rounded-xl border border-input-border shadow-xl"
        class:cursor-pointer={minimized}
        in:slide
        out:slide
        role="presentation"
        onclick={(e) => {
            e.stopPropagation();
            minimized = false;
        }}
    >
        <div class="flex gap-x-2 items-start justify-between">
            <div>
                <p class="font-medium">
                    Remaining {remaining} - Processed {uploadStore
                        .completedUploads.length}/{uploadStore.enqueuedUploads
                        .length + uploadStore.completedUploads.length}
                </p>
                <p class="text-sm">
                    Uploaded <span class="text-emerald-400"
                        >{successfulUploads}</span
                    >
                    - Error <span class="text-red-400">{errorUploads}</span> -
                    Duplicates
                    <span class="text-amber-500">{duplicateUploads}</span>
                </p>
            </div>
            <div class="space-x-2">
                <button title={$_('clear-all')} aria-label="Dismiss all" onclick={dismissAllCompleted}
                    ><i class="fa fa-ban"></i></button
                >
                <button
                    aria-label="Minimize"
                    onclick={(e) => {
                        e.stopPropagation();
                        minimized = true;
                    }}><i class="fa fa-minus"></i></button
                >
            </div>
        </div>
        <div
            class="max-h-96 max-w-72 mt-4 overflow-y-auto space-y-2"
            class:hidden={minimized}
        >
            {#each visibleUploads as u}
                <div class="bg-menu-item-background-hover rounded-lg py-2 px-3">
                    <div class="flex items-center gap-2">
                        <div class="w-6 shrink-0">
                            {#if u.status === "enqueued" || u.status == "uploading"}
                                <div class="spinner spinner-small"></div>
                            {:else}
                                <i
                                    class={{
                                        fa: true,
                                        "fa-circle-exclamation text-red-400":
                                            u.status == "error",
                                        "fa-triangle-exclamation text-amber-500":
                                            u.status == "duplicate",
                                        "fa-circle-check text-emerald-400":
                                            u.status == "success",
                                        "fa-ban text-gray-500":
                                            u.status == "cancelled",
                                    }}
                                ></i>
                            {/if}
                        </div>
                        <p class="text-xs basis-full min-w-0 break-all mr-2">
                            {u.file.name}
                        </p>
                        {#if u.status == "error" || u.status == "cancelled"}
                            <button
                                aria-label="Re-upload"
                                onclick={() => reUpload(u)}
                                ><i class="fa fa-redo text-sm"></i></button
                            >
                        {/if}
                        {#if u.status == "enqueued"}
                            <button
                                aria-label="Cancel upload"
                                onclick={() => cancelUpload(u)}
                                ><i class="fa fa-stop text-sm"></i></button
                            >
                        {/if}
                        {#if u.status == "duplicate"}
                            <button
                                title={$_("upload-despite-duplicate")}
                                aria-label={$_("upload-despite-duplicate")}
                                onclick={() => reUpload(u, true)}
                                ><i class="fa fa-upload text-sm"></i></button
                            >
                        {/if}
                        {#if u.status != "enqueued" && u.status != "uploading"}
                            <button
                                aria-label="Dismiss"
                                onclick={() => dismissUpload(u)}
                                ><i class="fa fa-close text-sm"></i></button
                            >
                        {/if}
                    </div>

                    {#if u.status == "uploading"}
                        <div
                            class="progress-bar my-1 rounded-md"
                            style="height:2px; width:{u.progress}%; background-color:#3549bb;transition: width 0.5s ease-in-out;"
                        ></div>
                    {:else if u.error}
                        <p class="text-red-400 text-xs">
                            {u.error}
                        </p>
                    {:else if u.duplicate}
                        <div class="text-amber-400 text-xs">
                            <p>
                                {$_("upload-duplicate-found")}:
                                <button
                                    class="underline text-left"
                                    onclick={() =>
                                        goto(`/trail/view/@${u.duplicate?.domain}/${u.duplicate!.id}`)}
                                    >{u.duplicate.name}</button
                                >
                            </p>
                            <p class="mt-1 font-medium break-words">
                                {#if u.duplicate.author && u.duplicate.author === $currentUser?.actor}
                                    {$_("upload-duplicate-own-trail")}
                                {:else}
                                    {$_("upload-duplicate-trail-owner", {
                                        values: { owner: u.duplicate.domain },
                                    })}
                                {/if}
                            </p>
                        </div>
                    {/if}
                </div>
            {/each}
        </div>
    </div>
{/if}
