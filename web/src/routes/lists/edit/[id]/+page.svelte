<script lang="ts">
    import { type List } from "$lib/models/list";
    import { _ } from "svelte-i18n";

    import { page } from "$app/state";
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import Button from "$lib/components/base/button.svelte";
    import Editor from "$lib/components/base/editor.svelte";
    import Search, {
        type SearchItem,
    } from "$lib/components/base/search.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import { ListCreateSchema } from "$lib/models/api/list_schema.js";
    import { TrailCreateSchema } from "$lib/models/api/trail_schema.js";
    import type { Trail, TrailSearchResult } from "$lib/models/trail.js";
    import { TrailShare } from "$lib/models/trail_share.js";
    import { lists_create, lists_update } from "$lib/stores/list_store.js";
    import { searchTrails } from "$lib/stores/search_store.js";
    import { theme } from "$lib/stores/theme_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        trail_share_create,
        trail_share_index,
    } from "$lib/stores/trail_share_store.js";
    import { trails_show, trails_update } from "$lib/stores/trail_store";
    import { currentUser } from "$lib/stores/user_store.js";
    import { getFileURL } from "$lib/util/file_util.js";
    import {
        formatDistance,
        formatElevation,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { z } from "zod";
    import { onMount } from "svelte";

    let { data } = $props();

    let previewURL = $state(data.previewUrl ?? "");
    let searchDropdownItems: SearchItem[] = $state([]);

    let activeTrailIndex: number = $state(-1);

    let map: MapWithElevationMaplibre | undefined = $state();

    let loading: boolean = $state(false);

    let newShares: TrailShare[] = [];

    let shareConfirmModal: ConfirmModal;
    let publishConfirmModal: ConfirmModal;

    let trailsOnMap: Trail[] = $state([]);

    const ClientListCreateSchema = ListCreateSchema.extend({
        _photos: z.array(z.instanceof(File)).optional(),
        avatar: z.string().or(z.instanceof(File)).optional(),
        expand: z
            .object({
                trails: z.array(TrailCreateSchema).optional(),
                list_share_via_list: z
                    .array(
                        z.object({
                            actor: z.string().length(15),
                            list: z.string().length(15),
                            permission: z.enum(["view", "edit"]),
                        }),
                    )
                    .optional(),
            })
            .optional(),
    });

    const {
        form,
        errors,
        data: formData,
    } = createForm<z.infer<typeof ClientListCreateSchema>>({
        initialValues: {
            ...data.list,
            public: data.list.id
                ? data.list.public
                : page.data.settings?.privacy?.lists === "public",
        },
        extend: validator({
            schema: ClientListCreateSchema,
        }),
        onSubmit: async (form) => {
            if (await checkPrerequisites()) {
                await saveList();
            }
        },
    });

    onMount(() => {
        trailsOnMap = [...(data.list.expand?.trails ?? [])];
    });

    async function checkPrerequisites() {
        if (
            (data.list?.public === false && $formData.public === true) ||
            ($formData.public === true &&
                (data.list?.expand?.trails?.length ?? 0) <
                    ($formData.expand?.trails?.length ?? 0))
        ) {
            publishConfirmModal.openModal();
            return false;
        } else if (await findNewTrailShares()) {
            shareConfirmModal.openModal();
            return false;
        }
        return true;
    }

    async function saveList() {
        const avatarFile = (
            document.getElementById("avatar") as HTMLInputElement
        ).files![0];
        loading = true;
        try {
            if ($formData.id) {
                await lists_update($formData as List, avatarFile);
            } else {
                const createdList = await lists_create(
                    $formData as List,
                    avatarFile,
                );
                $formData.id = createdList.id;
            }
            show_toast({
                type: "success",
                icon: "check",
                text: $_("list-saved-successfully"),
            });
        } catch (e) {
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-saving-list"),
            });
        } finally {
            loading = false;
        }
    }

    function openAvatarBrowser() {
        document.getElementById("avatar")!.click();
    }

    function handleAvatarSelection() {
        const files = (document.getElementById("avatar") as HTMLInputElement)
            .files;

        if (!files) {
            return;
        }

        previewURL = URL.createObjectURL(files[0]);
    }

    async function search(q: string) {
        try {
            const r = await searchTrails(q, {
                filter: `author = ${$currentUser?.actor} OR public = true`,
                sort: ["name:desc"],
                limit: 3,
            });

            searchDropdownItems = r
                .filter(
                    (h: TrailSearchResult) => !$formData.trails?.includes(h.id),
                )
                .map((t) => ({
                    text: t.name,
                    description: `${t.location ?? "-"}`,
                    value: t.id,
                    icon: "route",
                }));                
        } catch (e) {
            console.error(e);
        }
    }

    async function handleSearchClick(item: SearchItem) {
        const trail = await trails_show(item.value, undefined, undefined, true);
        $formData.trails?.push(trail.id!);
        $formData.expand = {
            trails: [...($formData.expand?.trails ?? []), trail],
            list_share_via_list: $formData.expand?.list_share_via_list,
        };

        trailsOnMap = [...($formData.expand?.trails ?? [])];
    }

    function deleteTrail(trail: Trail) {
        $formData.trails = $formData.trails?.filter((id) => id !== trail.id);
        $formData.expand!.trails = $formData.expand!.trails!.filter(
            (t) => t.id !== trail.id,
        );

        trailsOnMap = [...($formData.expand?.trails ?? [])];
    }

    async function findNewTrailShares() {
        const actorsWithAccess: string[] = [
            $formData.author!,
            ...($formData.expand?.list_share_via_list ?? []).map(
                (s) => s.actor!,
            ),
        ];
        for (const actorId of actorsWithAccess) {
            const existingTrailShares = await trail_share_index({
                actor: actorId,
            });
            for (const trail of $formData.expand?.trails ?? []) {
                if (
                    trail.author == actorId ||
                    trail.author != $currentUser?.id
                ) {
                    continue;
                }
                const trailShare = existingTrailShares.find(
                    (s) => s.trail == trail.id,
                );
                if (!trailShare) {
                    newShares.push(new TrailShare(actorId, trail.id!, "view"));
                }
            }
        }
        return newShares.length > 0;
    }

    async function updateTrailShares() {
        for (const newShare of newShares) {
            await trail_share_create(newShare);
        }
        newShares = [];
    }

    async function publishTrails() {
        for (const trail of $formData.expand?.trails ?? []) {
            if (trail.author !== $currentUser?.actor) {
                continue;
            }
            const updatedTrail: Trail = { ...trail, public: true };
            await trails_update(trail, updatedTrail);
        }
    }

    function moveTrail(trail: Trail, index: number, direction: 1 | -1) {
        (document?.activeElement as HTMLElement).blur();

        if (
            !$formData.trails ||
            !$formData.expand ||
            !$formData.expand.trails
        ) {
            return;
        }
        const previousTrail = $formData.expand?.trails?.at(index + direction);
        const previousTrailId = $formData.trails?.at(index + direction);

        if (!previousTrail || !previousTrailId) {
            return;
        }

        $formData.expand!.trails![index] = previousTrail;
        $formData.expand!.trails![index + direction] = trail;

        $formData.trails[index] = previousTrailId;
        $formData.trails[index + direction] = trail.id!;
    }
</script>

<svelte:head>
    <title
        >{$formData.id ? `${$formData.name} | ${$_("edit")}` : $_("new-list")} |
        wanderer</title
    >
</svelte:head>

<main class="grid grid-cols-1 md:grid-cols-[440px_1fr]">
    <form
        id="list-form"
        class="flex flex-col gap-4 px-8 order-1 md:order-none mt-8 md:mt-0 overflow-y-scroll"
        use:form
    >
        <h2 class="text-2xl font-semibold">
            {page.params.id === "new" ? $_("new-list") : $_("edit-list")}
        </h2>
        <label for="avatar" class="text-sm font-medium block">
            {$_("avatar")}
        </label>
        <input
            name="avatar"
            type="file"
            id="avatar"
            accept="image/*"
            style="display: none;"
            onchange={handleAvatarSelection}
        />
        <div class="flex items-center gap-4">
            {#if previewURL.length > 0}
                <img
                    class="w-32 aspect-square rounded-full object-cover border border-gray-100"
                    alt="avatar"
                    src={previewURL}
                />
            {:else}
                <div
                    class="w-32 aspect-square rounded-full bg-menu-item-background-focus border-gray-200"
                ></div>
            {/if}
            <button
                class="btn-secondary"
                type="button"
                onclick={openAvatarBrowser}>{$_("change")}...</button
            >
        </div>

        <TextField name="name" label={$_("name")} error={$errors.name}
        ></TextField>

        <Editor
            extraClasses="min-h-24"
            bind:value={$formData.description}
            label={$_("description")}
            error={$errors.description}
        ></Editor>
        <Toggle name="public" label={$_("public")}></Toggle>
        <h3 class="text-xl font-semibold">
            {$_("trail", { values: { n: 2 } })}
        </h3>
        <Search
            onupdate={search}
            onclick={handleSearchClick}
            placeholder="{$_('search-trails')}..."
            items={searchDropdownItems}
        ></Search>
        {#if $formData.expand?.trails?.length}
            {#each $formData.expand?.trails ?? [] as trail, i}
                <div
                    class="flex gap-4 p-4 rounded-xl border border-input-border cursor-pointer hover:bg-secondary-hover transition-colors items-center"
                    class:bg-secondary-hover={i == activeTrailIndex}
                    role="presentation"
                    onmouseenter={() => map?.highlightTrail(trail.id ?? "")}
                    onmouseleave={() => map?.unHighlightTrail(trail.id ?? "")}
                >
                    <div class="shrink-0">
                        <img
                            class="h-12 w-12 object-cover rounded-xl"
                            src={trail.photos.length
                                ? getFileURL(
                                      trail,
                                      trail.photos.at(trail.thumbnail ?? 0) ??
                                          trail.photos[0],
                                  )
                                : $theme === "light"
                                  ? emptyStateTrailLight
                                  : emptyStateTrailDark}
                            alt=""
                        />
                    </div>
                    <div class="basis-full">
                        <h4 class="font-semibold text-lg">
                            {trail.name}
                        </h4>
                        <span class="text-sm"
                            ><i class="fa fa-gauge mr-2"></i>{$_(
                                trail.difficulty ?? "?",
                            )}</span
                        >
                        <div
                            class="grid grid-cols-2 mt-1 gap-x-4 gap-y-2 text-sm text-gray-500"
                        >
                            <span
                                ><i class="fa fa-left-right mr-2"
                                ></i>{formatDistance(trail.distance)}</span
                            >
                            <span
                                ><i class="fa fa-clock mr-2"
                                ></i>{formatTimeHHMM(trail.duration)}</span
                            >
                            <span
                                ><i class="fa fa-arrow-trend-up mr-2"
                                ></i>{formatElevation(
                                    trail.elevation_gain,
                                )}</span
                            >
                            <span
                                ><i class="fa fa-arrow-trend-down mr-2"
                                ></i>{formatElevation(
                                    trail.elevation_loss,
                                )}</span
                            >
                        </div>
                    </div>
                    <div class="basis-0">
                        {#if i > 0}
                            <button
                                aria-label="Move up"
                                onclick={() => moveTrail(trail, i, -1)}
                                type="button"
                                class="btn-icon"
                            >
                                <i class="fa fa-chevron-up"></i>
                            </button>
                        {/if}
                        {#if i < $formData.expand?.trails?.length - 1}
                            <button
                                aria-label="Move down"
                                onclick={() => moveTrail(trail, i, 1)}
                                type="button"
                                class="btn-icon"
                            >
                                <i class="fa fa-chevron-down"></i>
                            </button>
                        {/if}
                        <button
                            aria-label="Remove trail"
                            type="button"
                            class="btn-icon text-red-500"
                            onclick={() => deleteTrail(trail)}
                            ><i class="fa fa-trash"></i></button
                        >
                    </div>
                </div>
            {/each}
        {:else}
            <span class="text-center text-sm text-gray-500 my-8"
                >{$_("no-routes-added")}</span
            >
        {/if}
        <Button
            primary={true}
            large={true}
            type="submit"
            extraClasses="mb-2"
            {loading}>{$_("save-list")}</Button
        >
    </form>
    <div id="trail-map" class="max-h-full">
        <MapWithElevationMaplibre
            fitBounds="animate"
            trails={trailsOnMap}
            bind:activeTrail={activeTrailIndex}
            bind:this={map}
        ></MapWithElevationMaplibre>
    </div>
</main>

<ConfirmModal
    id="share-confirm-modal"
    text={$_("list-share-warning-update")}
    title={$_("confirm-share")}
    action="confirm"
    bind:this={shareConfirmModal}
    onconfirm={async () => {
        await updateTrailShares();
        await saveList();
    }}
></ConfirmModal>

<ConfirmModal
    id="publish-confirm-modal"
    text={$_("list-public-warning")}
    title={$_("confirm-publish")}
    action="confirm"
    bind:this={publishConfirmModal}
    onconfirm={async () => {
        await publishTrails();
        await saveList();
    }}
></ConfirmModal>

<style>
    #trail-map {
        height: calc(50vh);
    }

    @media only screen and (min-width: 768px) {
        #trail-map,
        #list-form {
            height: calc(100vh - 124px);
        }
    }
</style>
