<script lang="ts">
    import { type Snippet } from "svelte";

    import { WaypointCreateSchema } from "$lib/models/api/waypoint_schema";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { waypoint } from "$lib/stores/waypoint_store";
    import { cloneDeep } from "$lib/util/deep_util";
    import { extractGPSCoordinates, type GPSCoordinates } from "$lib/util/exif_util";
    import { icons } from "$lib/util/icon_util";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";
    import Combobox from "../base/combobox.svelte";
    import Modal from "../base/modal.svelte";
    import TextField from "../base/text_field.svelte";
    import Textarea from "../base/textarea.svelte";
    import PhotoPicker from "../photo/photo_picker.svelte";
    import AssetPhotoPickerModal from "../trail/asset_photo_picker_modal.svelte";
    import type { Waypoint } from "$lib/models/waypoint";
    import type { PluginProvider } from "$lib/models/plugin_provider";
    import { photoLibraryCandidateKey, photoLibraryPluginLinks, type PhotoLibraryCandidate } from "$lib/models/photo_library";

    interface Props {
        children?: Snippet<[any]>;
        onsave?: (waypoint: Waypoint) => boolean | Promise<boolean> | void
        assetPluginActive?: boolean;
        assetPluginIds?: string[];
        assetPluginProviders?: PluginProvider[];
    }

    let { children, onsave, assetPluginIds = [], assetPluginProviders = [] }: Props = $props();

    let modal: Modal;
    let assetPhotoPickerModal: AssetPhotoPickerModal = $state()!;
    let pendingCandidates: PhotoLibraryCandidate[] = $state([]);

    export function closeModal() {
        modal.closeModal();
    }

    const ClientWaypointCreateSchema = WaypointCreateSchema.extend({
        photos: z.array(z.string()).default([]),
        _photos: z.array(z.instanceof(File)).optional(),
        _assetLinks: z.array(z.string()).optional(),
        _assetPluginLinks: z
            .array(
                z.object({
                    pluginId: z.string(),
                    assetIds: z.array(z.string()),
                }),
            )
            .optional(),
    });

    const { form, errors, data, setFields } = createForm<
        z.infer<typeof ClientWaypointCreateSchema>
    >({
        initialValues: $waypoint,
        extend: validator({ schema: ClientWaypointCreateSchema }),
        onSubmit: async (form) => {
            const wp = {
                ...(form as Waypoint),
                photos: $data.photos ?? [],
                _photos: $data._photos ?? [],
            } as Waypoint;
            if (pendingCandidates.length > 0) {
                const pluginCandidates = pendingCandidates.filter((c) => c.source !== "wanderer");
                const wandererCandidates = pendingCandidates.filter((c) => c.source === "wanderer");
                wp._assetCandidates = pluginCandidates.map((c) => ({
                    pluginId: c.pluginId ?? c.providerId ?? "",
                    assetId: c.assetId,
                    lat: c.lat,
                    lon: c.lon,
                    originalFileName: c.originalFileName,
                    takenAt: c.takenAt,
                }));
                wp._assetPluginLinks = photoLibraryPluginLinks(pluginCandidates);
                wp._assetLinks = wandererCandidates.map((c) => c.assetId);
                wp.photos = [
                    ...(wp.photos ?? []),
                    ...wandererCandidates
                        .map((candidate) => candidate.thumbnailUrl)
                        .filter((url): url is string => Boolean(url)),
                ];
            }
            const shouldClose = await onsave?.(wp);

            if (shouldClose !== false) {
                modal.closeModal!();
            }
        },
        transform: (values: unknown) => {
            const v = values as any;
            return {
                ...v,
                lat: parseFloat(v.lat),
                lon: parseFloat(v.lon),
            };
        },
    });

    $effect(() => {
        setFields(cloneDeep($waypoint));
        pendingCandidates = [];
    });

    let filteredIcons = $derived(
        ($data.icon?.length ?? 0) > 2
            ? icons
                  .filter((i) =>
                      i
                          .replaceAll("-", " ")
                          .includes($data.icon?.toLowerCase() ?? ""),
                  )
                  .map((i) => ({
                      text: i.replaceAll("-", " "),
                      value: i,
                      icon: i,
                  }))
            : [],
    );

    async function getCoordinatesFromPhoto(src: string, existingCoordinates?: GPSCoordinates) {
        const coordinates = existingCoordinates ?? (await extractGPSCoordinates(src));
        if (coordinates) {
            setFields("lat", coordinates.lat);
            setFields("lon", coordinates.lon);
            return;
        }
        show_toast({
            text: $_('no-gps-data-in-image'),
            icon: "close",
            type: "error",
        });
    }

    function onAssetPluginSelect(candidates: PhotoLibraryCandidate[]) {
        pendingCandidates = [...pendingCandidates, ...candidates.filter(
            (c) => !pendingCandidates.some((p) => candidateKey(p) === candidateKey(c))
        )];
    }

    function removePendingCandidate(assetId: string) {
        pendingCandidates = pendingCandidates.filter((c) => c.assetId !== assetId);
    }

    function candidateKey(candidate: PhotoLibraryCandidate): string {
        return photoLibraryCandidateKey(candidate);
    }

    const hasCoordinates = $derived(
        !isNaN(parseFloat(String($data.lat))) && !isNaN(parseFloat(String($data.lon)))
    );

    export function openModal() {
        modal.openModal();
    }

    const children_render = $derived(children);
</script>

<Modal
    id="waypoint-modal"
    size="md:min-w-2xl"
    title={$data.id ? $_("edit-waypoint") : $_("add-waypoint")}
    bind:this={modal}
>
    {#snippet children({ openModal })}
        {@render children_render?.({ openModal })}
    {/snippet}
    {#snippet content()}
        <form id="waypoint-form" class="modal-content space-y-4" use:form>
            <div class="flex gap-4">
                <div class="basis-2/3">
                    <TextField
                        name="name"
                        label={$_("name")}
                        error={$errors.name}
                    ></TextField>
                </div>

                <Combobox
                    name="icon"
                    icon={$data.icon}
                    bind:value={$data.icon}
                    items={filteredIcons}
                    label={$_("icon")}
                ></Combobox>
            </div>

            <Textarea
                name="description"
                label={$_("description")}
                error={$errors.description}
            ></Textarea>
            <div class="flex gap-4">
                <TextField name="lat" label={$_("latitude")} error={$errors.lat}
                ></TextField>
                <TextField
                    name="lon"
                    label={$_("longitude")}
                    error={$errors.lat}
                ></TextField>
            </div>
            <div>
                <label for="waypoint-photo-input" class="text-sm font-medium pb-1 block">
                    {$_("photos")}
                </label>
                <PhotoPicker
                    id="waypoint-photo-input"
                    parent={$data}
                    onexif={(src, coordinates) => getCoordinatesFromPhoto(src, coordinates)}
                    bind:photos={$data.photos}
                    bind:photoFiles={$data._photos}
                    showThumbnailControls={false}
                    showExifControls={true}
                    onassetplugin={hasCoordinates ? () => assetPhotoPickerModal.openModal() : undefined}
                    assetPluginPreviews={pendingCandidates.map(c => ({
                        pluginId: c.pluginId,
                        assetId: c.assetId,
                        filename: c.originalFileName,
                        takenAt: c.takenAt,
                        thumbnailUrl: c.thumbnailUrl,
                    }))}
                    onassetplugindelete={removePendingCandidate}
                ></PhotoPicker>
            </div>
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            <button class="btn-primary" type="submit" form="waypoint-form"
                >{$_("save")}</button
            >
        </div>
    {/snippet}
</Modal>

<AssetPhotoPickerModal
    bind:this={assetPhotoPickerModal}
    lat={parseFloat(String($data.lat)) || 0}
    lon={parseFloat(String($data.lon)) || 0}
    {assetPluginIds}
    {assetPluginProviders}
    onselect={onAssetPluginSelect}
/>
