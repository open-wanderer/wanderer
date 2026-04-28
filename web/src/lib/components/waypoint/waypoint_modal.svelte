<script lang="ts">
    import { type Snippet } from "svelte";

    import { WaypointCreateSchema } from "$lib/models/api/waypoint_schema";
    import { convertDMSToDD } from "$lib/models/gpx/utils";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { waypoint } from "$lib/stores/waypoint_store";
    import { cloneDeep } from "$lib/util/deep_util";
    import { icons } from "$lib/util/icon_util";
    import EXIF from "$lib/vendor/exif-js/exif";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";
    import Combobox from "../base/combobox.svelte";
    import Modal from "../base/modal.svelte";
    import TextField from "../base/text_field.svelte";
    import Textarea from "../base/textarea.svelte";
    import PhotoPicker from "../trail/photo_picker.svelte";
    import type { Waypoint } from "$lib/models/waypoint";

    interface Props {
        children?: Snippet<[any]>;
        onsave?: (waypoint: Waypoint) => boolean | Promise<boolean> | void
    }

    let { children, onsave }: Props = $props();

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    export function closeModal() {
        modal.closeModal();
    }

    const ClientWaypointCreateSchema = WaypointCreateSchema.extend({
        _photos: z.array(z.instanceof(File)).optional(),
    });

    const { form, errors, data, setFields } = createForm<
        z.infer<typeof ClientWaypointCreateSchema>
    >({
        initialValues: $waypoint,
        extend: validator({ schema: ClientWaypointCreateSchema }),
        onSubmit: async (form) => {
            const shouldClose = await onsave?.(form);

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

    function getCoordinatesFromPhoto(src: string) {
        EXIF.getData({ src: src }, function (p) {
            const lat = EXIF.getTag(p, "GPSLatitude");
            const latDir = EXIF.getTag(p, "GPSLatitudeRef");
            const lon = EXIF.getTag(p, "GPSLongitude");
            const lonDir = EXIF.getTag(p, "GPSLongitudeRef");

            if (lat && lon) {
                setFields("lat", convertDMSToDD(lat, latDir));
                setFields("lon", convertDMSToDD(lon, lonDir));
            } else {
                show_toast({
                    text: $_('no-gps-data-in-image'),
                    icon: "close",
                    type: "error",
                });
            }
        });
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
                <label
                    for="waypoint-photo-input"
                    class="text-sm font-medium pb-1"
                >
                    {$_("photos")}
                </label>
                <PhotoPicker
                    id="waypoint-photo-input"
                    parent={$data}
                    onexif={(src) => getCoordinatesFromPhoto(src)}
                    bind:photos={$data.photos}
                    bind:photoFiles={$data._photos}
                    showThumbnailControls={false}
                    showExifControls={true}
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
