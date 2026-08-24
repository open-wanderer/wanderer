<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import CodeEditor from "$lib/components/base/code_editor.svelte";
    import SingleSelect from "$lib/components/base/single_select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import type { RoutingProfile } from "$lib/models/routing";
    import { updateRoutingProfile } from "$lib/stores/routing_store.svelte";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { saveAs } from "$lib/util/file_util";
    import {
        decodeRoutingProfileText,
        encodeRoutingProfileText,
        routingProfileDownloadFilename,
    } from "$lib/util/routing_profile_util";
    import type { CodeEditorLanguageDefinition } from "$lib/util/code_editor_language";
    import { _ } from "svelte-i18n";

    interface Props {
        id: string;
        maxBytes?: number;
        fileExtension?: string;
        language?: CodeEditorLanguageDefinition;
        onsave?: (profile: RoutingProfile) => void;
    }

    let {
        id,
        maxBytes = 256 * 1024,
        fileExtension = ".txt",
        language,
        onsave,
    }: Props = $props();

    let modal: Modal;
    let profile: RoutingProfile | undefined = $state();
    let name = $state("");
    let mode: RoutingProfile["mode"] | "" = $state("");
    let profileContent = $state("");
    let originalProfileContent = $state("");
    let showModeFallback = $state(false);
    let saving = $state(false);
    let title = $derived(profile ? `${$_("edit")}: ${profile.name}` : $_("profile"));

    export function openModal(candidate: RoutingProfile) {
        if (!candidate.id || candidate.kind !== "custom_file") return;
        try {
            profileContent = decodeRoutingProfileText(candidate.contentBase64 ?? "");
        } catch (error) {
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
            return;
        }
        profile = candidate;
        name = candidate.name;
        mode = candidate.mode === "other" ? "" : candidate.mode;
        originalProfileContent = profileContent;
        showModeFallback = candidate.mode === "other";
        modal.openModal();
    }

    async function saveProfile() {
        if (
            !profile?.id ||
            !name.trim() ||
            !profileContent ||
            (showModeFallback && !mode) ||
            saving
        ) return;
        const contentBytes = new TextEncoder().encode(profileContent).length;
        if (contentBytes > maxBytes) {
            show_toast({
                text: $_("routing-profile-file-too-large", {
                    values: { size: formatBytes(maxBytes) },
                }),
                icon: "close",
                type: "error",
            });
            return;
        }

        saving = true;
        try {
            const metadata = { ...(profile.metadata ?? {}) };
            if (showModeFallback) {
                metadata.modeDetection = "manual";
            } else if (profileContent !== originalProfileContent) {
                delete metadata.modeDetection;
            }
            const saved = await updateRoutingProfile({
                ...profile,
                id: profile.id,
                name: name.trim(),
                mode: showModeFallback ? (mode as RoutingProfile["mode"]) : profile.mode,
                metadata,
                contentBase64: encodeRoutingProfileText(profileContent),
            });
            profile = saved;
            onsave?.(saved);
            if (saved.mode === "other") {
                mode = "";
                originalProfileContent = profileContent;
                showModeFallback = true;
                show_toast({
                    text: $_("routing-profile-mode-unresolved-help"),
                    icon: "triangle-exclamation",
                    type: "warning",
                });
                return;
            }
            modal.closeModal();
            show_toast({ text: $_("settings-saved"), icon: "check", type: "success" });
        } catch (error) {
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
        } finally {
            saving = false;
        }
    }

    function downloadProfile() {
        if (!profile) return;
        const metadata = profile.metadata ?? {};
        const filename = routingProfileDownloadFilename(
            metadata.filename,
            name || profile.name,
            fileExtension,
        );
        saveAs(
            new Blob([profileContent], {
                type: profile.contentType || "text/plain;charset=utf-8",
            }),
            filename,
        );
    }

    function modeItems() {
        return [
            { text: $_("routing-mode-foot"), value: "foot" },
            { text: $_("routing-mode-bike"), value: "bike" },
            { text: $_("routing-mode-motor"), value: "motor" },
        ];
    }

    function formatBytes(value: number) {
        return value >= 1024 ? `${Math.floor(value / 1024)} KB` : `${value} B`;
    }

    function errorMessage(error: unknown) {
        return error instanceof Error && error.message ? error.message : $_("error-generic");
    }
</script>

<Modal {id} {title} size="w-[min(94vw,64rem)] max-w-[94vw]" bind:this={modal}>
    {#snippet content()}
        <div
            class={`grid grid-cols-1 gap-3 ${showModeFallback
                ? "md:grid-cols-[minmax(0,1fr)_14rem]"
                : ""}`}
        >
            <TextField
                label={$_("name")}
                extraClasses="!h-10 !px-3 !py-0"
                bind:value={name}
            ></TextField>
            {#if showModeFallback}
                <div>
                    <SingleSelect
                        label={$_("routing-profile-mode")}
                        placeholder={$_("routing-profile-mode-select")}
                        items={modeItems()}
                        bind:value={mode}
                    ></SingleSelect>
                    <p class="mt-1 text-xs text-amber-400">
                        {$_("routing-profile-mode-unresolved-help")}
                    </p>
                </div>
            {/if}
        </div>
        <CodeEditor
            label={$_("routing-profile-content")}
            ariaLabel={$_("routing-profile-content")}
            {language}
            bind:value={profileContent}
        ></CodeEditor>
    {/snippet}

    {#snippet footer()}
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <button
                class="btn-secondary flex h-10 items-center justify-center gap-2 !py-0"
                type="button"
                disabled={!profile}
                onclick={downloadProfile}
            >
                <i class="fa fa-download"></i>
                <span>{$_("download")}</span>
            </button>
            <div class="grid grid-cols-2 gap-3 sm:flex sm:justify-end">
                <button
                    class="btn-secondary h-10 !py-0"
                    type="button"
                    disabled={saving}
                    onclick={() => modal.closeModal()}
                >
                    {$_("cancel")}
                </button>
                <button
                    class="btn-primary h-10 !min-h-0 !py-0"
                    class:btn-disabled={!name.trim() ||
                        !profileContent ||
                        (showModeFallback && !mode) ||
                        saving}
                    type="button"
                    disabled={!name.trim() ||
                        !profileContent ||
                        (showModeFallback && !mode) ||
                        saving}
                    onclick={saveProfile}
                >
                    {#if saving}<span class="spinner mr-2 inline-block h-4 w-4"></span>{/if}
                    {$_("save")}
                </button>
            </div>
        </div>
    {/snippet}
</Modal>
