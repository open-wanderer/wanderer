<script module lang="ts">
    export interface PhotoPickerFormApi {
        save: () => void;
        formPhotos: () => string[];
        storedPhotos: () => string[];
    }
</script>

<script lang="ts">
    import { createForm } from "felte";
    import { untrack } from "svelte";
    import { get, writable } from "svelte/store";
    import type { Trail } from "#lib/models/trail.ts";
    import { copyTrailEditorValues } from "#lib/util/trail_editor_util.ts";
    import PhotoPicker from "./photo_picker.svelte";

    const props: {
        api: PhotoPickerFormApi;
        saved: Trail;
    } = $props();
    const fixture = untrack(() => props);
    const stored = writable(fixture.saved);
    const { form, data: formData, setFields } = createForm<Pick<Trail, "id" | "photos" | "thumbnail">>({
        initialValues: copyTrailEditorValues(fixture.saved),
        onSubmit: () => {},
    });
    fixture.api.save = () => {
        stored.set(fixture.saved);
        setFields(fixture.saved);
    };
    fixture.api.formPhotos = () => get(formData).photos;
    fixture.api.storedPhotos = () => get(stored).photos;
</script>

<form use:form>
    <PhotoPicker id="trail" parent={$formData} bind:photos={$formData.photos} bind:thumbnail={$formData.thumbnail} photoFiles={undefined} />
</form>
