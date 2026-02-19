<script lang="ts">
    import type { Trail } from "$lib/models/trail";
    import { TrailLike } from "$lib/models/trail_like";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import {
        trail_like_create,
        trail_like_delete,
    } from "$lib/stores/trail_like_store";
    import { currentUser } from "$lib/stores/user_store";
    import confetti from "canvas-confetti";
    import { untrack } from "svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        trail: Trail;
    }

    let likeButton: HTMLButtonElement;

    let { trail }: Props = $props();

    let likeCount = $state(untrack(() => trail.like_count));

    let loading = $state(false);

    async function like(e: Event) {
        e.stopPropagation();
        e.preventDefault();
        if (!$currentUser || !trail.id) {
            return;
        }

        loading = true;
        try {
            if (trailLike) {
                await trail_like_delete(trailLike);
                likeCount -= 1;
                trailLike = undefined;
            } else {
                trailLike = await trail_like_create(
                    new TrailLike($currentUser.actor, trail.id),
                );
                likeCount += 1;

                const rect = likeButton.getBoundingClientRect();
                const x = (rect.left + rect.width / 2) / window.innerWidth;
                const y = (rect.top + rect.height / 2) / window.innerHeight;

                confetti({
                    particleCount: 60,
                    startVelocity: 20,
                    spread: 40,
                    origin: { x, y },
                    colors: ["#fb2c36"],
                    scalar: 1.2,
                });
            }
        } catch (e) {
            show_toast({
                icon: "close",
                text: $_("error-liking-trail"),
                type: "error",
            });
        } finally {
            loading = false;
        }
    }

    let trailLike = $state(
        untrack(() =>
            trail.expand?.trail_like_via_trail?.find(
                (l) => l.actor === $currentUser?.actor,
            ),
        ),
    );
</script>

<button
    class="like-button rounded-full w-12 py-2 bg-background {$currentUser ? 'hover:bg-secondary-hover' : ''} transition-colors relative"
    aria-label="like"
    type="button"
    bind:this={likeButton}
    disabled={!$currentUser}
    onclick={like}
    data-title={$_("likes")}
>
    <i
        class="{trailLike ? 'fa' : 'fa-regular'} fa-heart text-3xl"
        class:text-red-500={trailLike !== undefined}
        class:loading
    ></i>
    <p class="font-semibold">{likeCount}</p>
</button>

<style>
    .like-button > i.loading {
        animation: heartbeat 1s infinite;
    }

    @keyframes heartbeat {
        0%,
        100% {
            transform: scale(1);
        }
        20% {
            transform: scale(1.1);
        }
        40% {
            transform: scale(0.95);
        }
        60% {
            transform: scale(1.05);
        }
    }
</style>
