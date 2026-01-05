<script lang="ts">
    import { type Snippet } from "svelte";

    import { SummitLogCreateSchema } from "$lib/models/api/summit_log_schema";
    import GPX from "$lib/models/gpx/gpx";
    import { summitLog } from "$lib/stores/summit_log_store";
    import { fetchGPX } from "$lib/stores/trail_store";
    import { cloneDeep } from "$lib/util/deep_util";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";
    import Datepicker from "../base/datepicker.svelte";
    import Modal from "../base/modal.svelte";
    import Textarea from "../base/textarea.svelte";
    import PhotoPicker from "../trail/photo_picker.svelte";
    import TrailPicker from "../trail/trail_picker.svelte";
    import type { SummitLog } from "$lib/models/summit_log";
    import Editor from "../base/editor.svelte";
    interface Props {
        children?: Snippet<[any]>;
        onsave?: (summitLog: SummitLog) => void;
    }

    let { children, onsave }: Props = $props();

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    const ClientSummitLogCreateSchema = SummitLogCreateSchema.extend({
        _photos: z.array(z.instanceof(File)).optional(),
        _gpx: z.instanceof(File).optional().nullable(),
        expand: z
            .object({
                gpx_data: z.string().optional(),
            })
            .optional(),
    });

    const { form, errors, data, setFields } = createForm<
        z.infer<typeof ClientSummitLogCreateSchema>
    >({
        initialValues: $summitLog,
        extend: validator({ schema: ClientSummitLogCreateSchema }),
        onSubmit: async (form) => {
            if (
                !form._photos?.length &&
                !form.photos?.length &&
                form.expand?.gpx_data
            ) {
                const canvas = document.querySelector(
                    "#trail-picker-map .maplibregl-canvas",
                ) as HTMLCanvasElement;

                const dataURL = canvas.toDataURL();
                const response = await fetch(dataURL);
                const blob = await response.blob();
                form._photos = [new File([blob], "route")];
            }

            onsave?.(form);
            modal.closeModal!();
        },
    });

    $effect(() => {
        setFields(cloneDeep($summitLog));
    });

    $effect(() => {
        if ($summitLog._gpx) {
            $data._gpx = $summitLog._gpx;
        }
    });

    let gpxLoading = $state(false);

    async function ensureGpxDataLoaded() {
        if (gpxLoading || !$data.id || !$data.gpx || $data.expand?.gpx_data) {
            return;
        }
        gpxLoading = true;
        try {
            const gpxData = await fetchGPX($data as any, fetch);
            if (!gpxData) {
                return;
            }
            if (!$data.expand) {
                $data.expand = {};
            }
            $data.expand.gpx_data = gpxData;
        } finally {
            gpxLoading = false;
        }
    }

    $effect(() => {
        void ensureGpxDataLoaded();
    });

    async function handleTrailSelection(trailData: string | null) {
        if (!trailData) {
            $data.duration = undefined;
            $data.elevation_gain = undefined;
            $data.elevation_loss = undefined;
            $data.distance = undefined;
            $data.gpx = "";
            if ($data.expand) {
                $data.expand.gpx_data = undefined;
            }
            $data._gpx = null;
            return;
        }
        const gpxObject = GPX.parse(trailData);
        try {
            await gpxObject.correctElevation();
        } catch (e) {
            console.warn("Unable to correct elevation: " + e);
        }

        const totals = gpxObject.features;
        $data.duration = totals.duration / 1000;
        $data.elevation_gain = totals.elevationGain;
        $data.elevation_loss = totals.elevationLoss;
        $data.distance = totals.distance;
        $data.expand!.gpx_data = trailData;
    }

    const children_render = $derived(children);
</script>

<Modal
    id="summit-log-modal"
    title={$data.id ? $_("edit-entry") : $_("add-entry")}
    size="md:min-w-2xl"
    bind:this={modal}
>
    {#snippet children({ openModal })}
        {@render children_render?.({ openModal })}
    {/snippet}
    {#snippet content()}
        <form id="summit-log-form" class="modal-content space-y-4" use:form>
            <div class="flex">
                <Datepicker name="date" label={$_("date")} error={$errors.date}
                ></Datepicker>
            </div>
            <div>
                <label
                    for="summitlog-photo-input"
                    class="text-sm font-medium pb-1"
                >
                    {$_("photos")}
                </label>
                <PhotoPicker
                    id="summitlog-photo-input"
                    parent={$data}
                    bind:photos={$data.photos}
                    bind:photoFiles={$data._photos}
                    showThumbnailControls={false}
                ></PhotoPicker>
            </div>
            <div class="flex gap-4 items-end">
                {#if $data.expand}
                    <TrailPicker
                        bind:trailFile={$data._gpx}
                        bind:trailData={$data.expand.gpx_data}
                        hasTrail={Boolean($data.gpx)}
                        label={$_("trail", { values: { n: 1 } })}
                        onchange={(trail) => handleTrailSelection(trail)}
                    ></TrailPicker>
                {/if}
                <div class="basis-full">
                    <Editor
                        bind:value={$data.text}
                        extraClasses="min-h-28"
                        label={$_("text")}
                        error={$errors.text}
                        searchListPosition="fixed"
                    ></Editor>
                </div>
            </div>
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            <button class="btn-primary" type="submit" form="summit-log-form"
                >{$_("save")}</button
            >
        </div>
    {/snippet}
</Modal>
