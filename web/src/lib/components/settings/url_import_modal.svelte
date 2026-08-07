<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { trails_upload } from "$lib/stores/trail_store";
    import {
        processUploadQueue,
        uploadStore,
        type Upload,
    } from "$lib/stores/upload_store.svelte";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";
    import TextField from "../base/text_field.svelte";
    import Button from "../base/button.svelte";

    let modal: Modal;

    let loading: boolean = false;

    export function openModal() {
        setFields("url", "");
        setErrors("url", []);
        loading = false;
        modal.openModal();
    }

    const { form, errors, setFields, setErrors } = createForm<{
        url: string;
    }>({
        initialValues: { url: "" },
        extend: validator({
            schema: z.object({
                url: z.string().min(1, "required").url("not-a-valid-url"),
            }),
        }),
        onSubmit: async (form) => {
            try {
                loading = true;
                const r = await fetch("/api/v1/trail/download", {
                    method: "POST",
                    body: JSON.stringify({ url: form.url }),
                });
                if (!r.ok) {
                    throw new Error(`Failed to fetch file: ${r.statusText}`);
                }

                const blob = await r.blob();

                const fileName =
                    form.url.split("/").pop() ||
                    "url-import-file-" + new Date().getTime();

                const file = new File([blob], fileName, { type: blob.type });

                const u: Upload = {
                    file: file,
                    progress: 0,
                    status: "enqueued",
                    function: trails_upload,
                };
                uploadStore.enqueuedUploads.push(u);

                await processUploadQueue();
                modal.closeModal();
            } catch (error) {
                console.error("Error:", error);
            } finally {
                loading = false;
            }
        },
    });
</script>

<Modal
    id="url-import-modal"
    size="md:min-w-sm"
    title={$_("from-url")}
    bind:this={modal}
>
    {#snippet content()}
        <form id="url-import-form" use:form>
            <TextField
                name="url"
                label="URL"
                placeholder="https://..."
                error={$errors.url}
            ></TextField>
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button
                type="button"
                class="btn-secondary"
                onclick={() => modal.closeModal()}>{$_("cancel")}</button
            >
            <Button primary type="submit" form="url-import-form" {loading}
                >{$_("import")}</Button
            >
        </div>
    {/snippet}</Modal
>
