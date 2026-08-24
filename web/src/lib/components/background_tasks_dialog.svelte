<script lang="ts">
    import {
        backgroundTaskStore,
        removeBackgroundTask,
        type BackgroundTask,
    } from "$lib/stores/background_task_store.svelte";
    import { goto } from "$app/navigation";
    import { slide } from "svelte/transition";
    import { _ } from "svelte-i18n";

    let minimized: boolean = $state(false);

    let remaining = $derived(
        backgroundTaskStore.tasks.filter((task) => task.status === "enqueued" || task.status === "running").length,
    );

    let processed = $derived(backgroundTaskStore.tasks.length - remaining);

    let hasCompleted = $derived(
        backgroundTaskStore.tasks.some((task) => task.status !== "enqueued" && task.status !== "running"),
    );

    function dismissTask(task: BackgroundTask) {
        task.onDismiss?.();
        removeBackgroundTask(task.id);
    }

    function dismissAllCompleted() {
        for (const task of [...backgroundTaskStore.tasks]) {
            if (task.status !== "enqueued" && task.status !== "running") {
                dismissTask(task);
            }
        }
    }

    async function runAction(task: BackgroundTask, action: NonNullable<BackgroundTask["actions"]>[number]) {
        if (action.href) {
            await goto(action.href);
            return;
        }
        await action.run?.();
    }

    function iconClass(task: BackgroundTask) {
        return {
            fa: true,
            "fa-circle-exclamation text-red-400": task.status === "error",
            "fa-triangle-exclamation text-amber-500": task.status === "warning",
            "fa-circle-check text-emerald-400": task.status === "success",
            "fa-ban text-gray-500": task.status === "cancelled",
        };
    }
</script>

{#if backgroundTaskStore.tasks.length}
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
                    {$_("background-tasks-summary", {
                        values: {
                            remaining,
                            processed,
                            total: backgroundTaskStore.tasks.length,
                        },
                    })}
                </p>
            </div>
            <div class="space-x-2">
                {#if hasCompleted}
                    <button
                        title={$_("background-tasks-dismiss-completed")}
                        aria-label={$_("background-tasks-dismiss-completed")}
                        onclick={dismissAllCompleted}
                        ><i class="fa fa-ban"></i></button
                    >
                {/if}
                <button
                    aria-label={$_("minimize")}
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
            {#each backgroundTaskStore.tasks as task (task.id)}
                <div class="bg-menu-item-background-hover rounded-lg py-2 px-3">
                    <div class="flex items-center gap-2">
                        <div class="w-6 shrink-0">
                            {#if task.status === "enqueued" || task.status === "running"}
                                <div class="spinner spinner-small"></div>
                            {:else}
                                <i class={iconClass(task)}></i>
                            {/if}
                        </div>
                        <p class="text-xs basis-full min-w-0 break-all mr-2">
                            {task.title}
                        </p>
                        {#each task.actions ?? [] as action}
                            <button
                                title={action.label}
                                aria-label={action.label}
                                onclick={() => runAction(task, action)}
                                ><i class={`fa fa-${action.icon} text-sm`}></i></button
                            >
                        {/each}
                        {#if task.status !== "enqueued" && task.status !== "running"}
                            <button
                                aria-label={$_("dismiss")}
                                onclick={() => dismissTask(task)}
                                ><i class="fa fa-close text-sm"></i></button
                            >
                        {/if}
                    </div>

                    {#if task.status === "running"}
                        <div
                            class="progress-bar my-1 rounded-md"
                            style="height:2px; width:{task.progress ?? 0}%; background-color:#3549bb;transition: width 0.5s ease-in-out;"
                        ></div>
                    {/if}
                    {#if task.detail}
                        <p
                            class={{
                                "text-xs": true,
                                "text-red-400": task.status === "error",
                                "text-amber-400": task.status === "warning",
                                "text-content/75": task.status !== "error" && task.status !== "warning",
                            }}
                        >
                            {task.detail}
                        </p>
                    {/if}
                </div>
            {/each}
        </div>
    </div>
{/if}
