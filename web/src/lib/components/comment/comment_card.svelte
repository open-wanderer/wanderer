<script lang="ts">
    import type { Comment } from "$lib/models/comment";
    import { formatTimeSince } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import { untrack } from "svelte";
    import { fade } from "svelte/transition";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import Editor from "../base/editor.svelte";

    interface Props {
        comment: Comment;
        mode?: "show" | "edit";
        ondelete?: (comment: Comment) => void;
        onedit?: (data: { comment: Comment; text: string }) => void;
    }

    let { comment, mode = "show", ondelete, onedit }: Props = $props();

    let editing: boolean = $state(false);

    let editedComment = $state(untrack(() => comment.text));

    $effect(() => {
        editedComment = comment.text;
    });

    let avatarSrc = $derived(
        comment.expand?.author.icon
            ? comment.expand?.author.icon
            : `https://api.dicebear.com/7.x/initials/svg?seed=${comment.expand?.author.preferred_username ?? ""}&backgroundType=gradientLinear`,
    );

    let timeSince = $derived(formatTimeSince(new Date(comment.created ?? "")));

    function deleteComment() {
        ondelete?.(comment);
    }

    function toggleEdit() {
        if (editing) {
            onedit?.({ comment: comment, text: editedComment });
        }
        editing = !editing;
    }
</script>

<div
    class="flex gap-4 items-start"
    in:fade={{ duration: 150 }}
    out:fade={{ duration: 150 }}
>
    <a
        href="/profile/{handleFromRecordWithIRI(comment)}"
        class="text-sm font-semibold shrink-0"
    >
        <img
            class="rounded-full w-10 aspect-square"
            src={avatarSrc}
            alt="avatar"
        />
    </a>
    <div>
        <div class="flex items-center">
            <p class="">
                <a
                    href="/profile/{handleFromRecordWithIRI(comment)}"
                    class="text-sm font-semibold"
                    >{handleFromRecordWithIRI(comment)}</a
                >
                <span class="text-xs text-gray-500 ml-2"
                    >{$_(`n-${timeSince.unit}-ago`, {
                        values: { n: timeSince.value },
                    })}
                    {comment.updated != comment.created
                        ? `(${$_("edited")})`
                        : ""}</span
                >
            </p>
            {#if mode == "edit"}
                <button
                    aria-label="Edit comment"
                    type="button"
                    class="btn-icon ml-2"
                    style="font-size: 0.75rem;"
                    onclick={toggleEdit}
                    ><i class="fa fa-{editing ? 'check' : 'pen'}"></i></button
                >
                <button
                    aria-label="Delete comment"
                    type="button"
                    class="btn-icon text-xs"
                    style="font-size: 0.75rem;"
                    onclick={() => {
                        if (editing) {
                            editing = false;
                        } else {
                            deleteComment();
                        }
                    }}><i class="fa {editing ? 'fa-close' : 'fa-trash'}"></i></button
                >
            {/if}
        </div>
        {#if editing}
            <Editor extraClasses="mt-2" bind:value={editedComment}></Editor>
        {:else}
            {@html comment.text}
        {/if}
    </div>
</div>
