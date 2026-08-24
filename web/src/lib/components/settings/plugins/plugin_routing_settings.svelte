<script lang="ts">
    import FiniteNumberField from "$lib/components/base/finite_number_field.svelte";
    import SingleSelect from "$lib/components/base/single_select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import RoutingProfileEditorModal from "$lib/components/settings/plugins/routing_profile_editor_modal.svelte";
    import type { Category } from "$lib/models/category";
    import type { PluginInstance } from "$lib/models/plugin_instance";
    import type { PluginProvider } from "$lib/models/plugin_provider";
    import type { Subcategory } from "$lib/models/subcategory";
    import type {
        RoutingControl,
        RoutingNativeControls,
        RoutingProfile,
        RoutingProfileMapping,
        RoutingSettings,
    } from "$lib/models/routing";
    import { categoryPreferences } from "$lib/stores/category_preference_store";
    import {
        createRoutingProfile,
        createRoutingProfileMapping,
        deleteRoutingProfile,
        routingNativeControls,
        routingProfileMappings,
        routingProfiles,
        routingSettings,
        updateRoutingProfile,
        updateRoutingProfileMapping,
    } from "$lib/stores/routing_store.svelte";
    import { subcategoryPreferences } from "$lib/stores/subcategory_preference_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import {
        displayCategoryIcon,
        displayCategoryName,
        displaySubcategoryBadgeIcon,
        displaySubcategoryIcon,
        displaySubcategoryLabel,
        displaySubcategoryShortBadge,
        preferenceForCategory,
        sortedCategoriesByPreference,
        sortedSubcategoriesByPreference,
        subcategoryVisible,
    } from "$lib/util/category_util";
    import { localizedText } from "$lib/util/plugin_i18n";
    import { codeEditorLanguageDefinition } from "$lib/util/code_editor_language";
    import { saveAs } from "$lib/util/file_util";
    import {
        decodeRoutingProfileText,
        routingProfileDownloadFilename,
    } from "$lib/util/routing_profile_util";
    import { _, locale } from "svelte-i18n";

    interface RoutingCategoryRow {
        key: string;
        category: Category;
        subcategory?: Subcategory;
    }

    interface RoutingCategoryGroup {
        category: Category;
        row: RoutingCategoryRow;
        children: RoutingCategoryRow[];
    }

    interface Props {
        plugin: PluginProvider;
        categories?: Category[];
        subcategories?: Subcategory[];
        instance?: PluginInstance;
    }

    let { plugin, categories = [], subcategories = [], instance }: Props = $props();

    let settings: RoutingSettings = $state({});
    let profiles: RoutingProfile[] = $state([]);
    let mappings: RoutingProfileMapping[] = $state([]);
    let selections: Record<string, string> = $state({});
    let nativeConfigs: Record<string, Record<string, unknown>> = $state({});
    let dirtyMappings: Record<string, boolean> = $state({});
    let nativeControlsByMapping: Record<string, RoutingNativeControls | undefined> = $state({});
    let advancedMapping = $state("");
    let expandedCategories: Record<string, boolean> = $state({});
    let loading = $state(false);
    let loadError = $state("");
    let uploading = $state(false);
    let profileUploadInput: HTMLInputElement | undefined = $state();
    let profileEditorModal: RoutingProfileEditorModal | undefined = $state();
    let profileDeleteModal: ConfirmModal | undefined = $state();
    let profileToDelete: RoutingProfile | undefined = $state();

    let categoryGroups = $derived.by(() =>
        sortedCategoriesByPreference(categories, $categoryPreferences, $locale)
            .filter(
                (category) =>
                    preferenceForCategory($categoryPreferences, category.id)?.visible !== false,
            )
            .map(
                (category): RoutingCategoryGroup => ({
                    category,
                    row: { key: mappingKey(category.name), category },
                    children: sortedSubcategoriesByPreference(
                        subcategories.filter(
                            (subcategory) =>
                                subcategory.category === category.id &&
                                subcategoryVisible(subcategory.id, $subcategoryPreferences),
                        ),
                        $subcategoryPreferences,
                        $locale,
                    ).map((subcategory) => ({
                        key: mappingKey(category.name, subcategory.name),
                        category,
                        subcategory,
                    })),
                }),
            ),
    );
    let categoryRows = $derived(
        categoryGroups.flatMap((group) => [group.row, ...group.children]),
    );
    let pluginProfiles = $derived(
        profiles.filter(
            (profile) =>
                profile.pluginId === plugin.id &&
                profile.enabled &&
                !(profile.kind === "custom_file" && profile.mode === "other"),
        ),
    );
    let routingMetadata = $derived(
        objectValue((plugin.metadata as Record<string, unknown> | undefined)?.routing),
    );
    let uploadMetadata = $derived(objectValue(routingMetadata.nativeProfileUpload));
    let profileUploadAvailable = $derived(
        featureEnabled("profileUpload") && uploadMetadata.enabled === true,
    );
    let profileUploadMaxBytes = $derived.by(() => {
        const configured = Number(uploadMetadata.maxBytes ?? 256 * 1024);
        return Number.isFinite(configured) && configured > 0 ? configured : 256 * 1024;
    });
    let profileEditorLanguage = $derived(
        codeEditorLanguageDefinition(uploadMetadata.editorLanguage),
    );
    let profileFileExtension = $derived.by(() => {
        const extensions = uploadMetadata.extensions;
        if (!Array.isArray(extensions)) return ".txt";
        const extension = extensions.find((value): value is string => typeof value === "string");
        return extension || ".txt";
    });
    let nativeControlsAvailable = $derived(featureEnabled("nativeAdvancedControls"));
    let customProfiles = $derived.by(() =>
        profiles
            .filter((profile) => profile.pluginId === plugin.id && profile.scope === "user")
            .sort((left, right) =>
                profileLabel(left).localeCompare(profileLabel(right), $locale ?? undefined, {
                    sensitivity: "base",
                }),
            ),
    );
    let profileEditorModalId = $derived(
        `routing-profile-editor-${domKey(plugin.id)}-${domKey(instance?.id ?? "default")}`,
    );

    export async function open() {
        loading = true;
        loadError = "";
        advancedMapping = "";
        expandedCategories = {};
        try {
            const [loadedSettings, loadedProfiles, loadedMappings] = await Promise.all([
                routingSettings(),
                routingProfiles(),
                routingProfileMappings(),
            ]);
            settings = loadedSettings;
            profiles = loadedProfiles;
            mappings = loadedMappings;
            dirtyMappings = {};
            nativeControlsByMapping = {};
            initializeMappings();
        } catch (error) {
            loadError = errorMessage(error);
        } finally {
            loading = false;
        }
    }

    export async function save() {
        if (loading) {
            throw new Error($_("routing-settings-loading"));
        }
        if (loadError) {
            throw new Error(loadError);
        }
        for (const row of categoryRows) {
            if (!dirtyMappings[row.key]) continue;
            await saveMapping(row);
        }
        dirtyMappings = {};
    }

    function initializeMappings() {
        const nextSelections: Record<string, string> = {};
        const nextNativeConfigs: Record<string, Record<string, unknown>> = {};
        for (const row of categoryRows) {
            const mapping = effectiveMapping(row);
            const selection = mappingSelection(mapping);
            nextSelections[row.key] = selection;
            const profile = profileForSelection(selection);
            nextNativeConfigs[row.key] = deepMerge(
                profileNativeConfig(profile),
                cloneObject(mapping?.nativeConfig),
            );
        }
        selections = nextSelections;
        nativeConfigs = nextNativeConfigs;
    }

    function effectiveMapping(row: RoutingCategoryRow) {
        return mappings
            .filter(
                (mapping) =>
                    mapping.pluginId === plugin.id &&
                    mapping.category === row.category.name &&
                    (!mapping.subcategory || mapping.subcategory === row.subcategory?.name) &&
                    (!mapping.instanceId || mapping.instanceId === instance?.id),
            )
            .sort((left, right) => mappingRank(left) - mappingRank(right))
            .at(-1);
    }

    function mappingRank(mapping: RoutingProfileMapping) {
        const scopeRank = mapping.scope === "user" ? 200 : mapping.scope === "admin" ? 100 : 0;
        return scopeRank + (mapping.instanceId ? 10 : 0) + (mapping.subcategory ? 1 : 0);
    }

    function editableMapping(row: RoutingCategoryRow) {
        return mappings
            .filter(
                (mapping) =>
                    mapping.scope === "user" &&
                    mapping.pluginId === plugin.id &&
                    mapping.category === row.category.name &&
                    (mapping.subcategory ?? "") === (row.subcategory?.name ?? "") &&
                    (mapping.instanceId ?? "") === (instance?.id ?? ""),
            )
            .sort((left, right) => mappingRank(left) - mappingRank(right))
            .at(-1);
    }

    function mappingSelection(mapping?: RoutingProfileMapping) {
        if (mapping?.profileId) return `profile:${mapping.profileId}`;
        if (mapping?.nativeProfileKey) return `native:${mapping.nativeProfileKey}`;
        return "";
    }

    function profileForSelection(selection: string) {
        if (selection.startsWith("profile:")) {
            const id = selection.slice("profile:".length);
            return profiles.find((profile) => profile.id === id);
        }
        if (selection.startsWith("native:")) {
            const key = selection.slice("native:".length);
            return profiles.find(
                (profile) =>
                    profile.pluginId === plugin.id &&
                    profile.kind === "builtin" &&
                    profile.key === key,
            );
        }
        return undefined;
    }

    function profileItems() {
        return pluginProfiles
            .map((profile) => ({
                text: profileLabel(profile),
                value: profile.id ? `profile:${profile.id}` : `native:${profile.key}`,
                icon: profile.source === "user" ? "fa-user" : undefined,
            }))
            .sort((left, right) =>
                left.text.localeCompare(right.text, $locale ?? undefined, {
                    sensitivity: "base",
                }),
            );
    }

    function profileLabel(profile: RoutingProfile) {
        return metadataLabel(objectValue(profile.metadata).labels, profile.name);
    }

    function categoryLabel(row: RoutingCategoryRow) {
        return row.subcategory
            ? displaySubcategoryLabel(row.subcategory, $locale)
            : displayCategoryName(row.category, $locale);
    }

    function categoryIcon(row: RoutingCategoryRow) {
        return row.subcategory
            ? displaySubcategoryIcon(row.subcategory, row.category)
            : displayCategoryIcon(row.category);
    }

    function categoryBadgeIcon(row: RoutingCategoryRow) {
        return row.subcategory ? displaySubcategoryBadgeIcon(row.subcategory) : "";
    }

    function categoryBadge(row: RoutingCategoryRow) {
        return row.subcategory
            ? displaySubcategoryShortBadge(row.subcategory, $locale)
            : "";
    }

    function isCategoryExpanded(category: Category) {
        return expandedCategories[category.id] ?? false;
    }

    function toggleCategoryExpanded(category: Category) {
        expandedCategories = {
            ...expandedCategories,
            [category.id]: !isCategoryExpanded(category),
        };
    }

    function featureEnabled(key: string) {
        return settings.exposedFeatures?.[key] !== false;
    }

    function selectProfile(row: RoutingCategoryRow, selection: string) {
        const nextSelections = { ...selections, [row.key]: selection };
        const nextNativeConfigs = {
            ...nativeConfigs,
            [row.key]: profileNativeConfig(profileForSelection(selection)),
        };
        const nextNativeControls = { ...nativeControlsByMapping, [row.key]: undefined };
        for (const child of inheritedChildren(row)) {
            nextSelections[child.key] = selection;
            nextNativeConfigs[child.key] = profileNativeConfig(profileForSelection(selection));
            nextNativeControls[child.key] = undefined;
        }
        selections = nextSelections;
        nativeConfigs = nextNativeConfigs;
        nativeControlsByMapping = nextNativeControls;
        dirtyMappings = { ...dirtyMappings, [row.key]: true };
        if (advancedMapping === row.key) {
            if (profileSupportsNativeControls(row.key)) {
                void loadNativeControls(row.key);
            } else {
                advancedMapping = "";
            }
        }
    }

    function profileSupportsNativeControls(mappingKey: string) {
        if (!nativeControlsAvailable) return false;
        const profile = profileForSelection(selections[mappingKey] ?? "");
        if (!profile) return false;
        const metadata = objectValue(profile.metadata);
        if (metadata.advanced === false) return false;
        if (metadata.advanced === true) return true;
        if (Array.isArray(metadata.nativeControlGroups) && metadata.nativeControlGroups.length > 0) {
            return true;
        }
        return (
            profile.kind === "custom_file" &&
            objectValue(routingMetadata.nativeControlDiscovery).enabled === true
        );
    }

    async function saveMapping(row: RoutingCategoryRow) {
        const selection = selections[row.key] ?? "";
        if (!selection) return;
        const existing = editableMapping(row);
        const submitted: Partial<RoutingProfileMapping> & { id?: string } = {
            scope: "user",
            category: row.category.name,
            subcategory: row.subcategory?.name ?? "",
            pluginId: plugin.id,
            instanceId: instance?.id,
            preferences: cloneObject(existing?.preferences),
            nativeConfig: cloneObject(nativeConfigs[row.key]),
        };
        if (selection.startsWith("profile:")) {
            submitted.profileId = selection.slice("profile:".length);
            submitted.nativeProfileKey = "";
        } else {
            submitted.nativeProfileKey = selection.slice("native:".length);
            submitted.profileId = "";
        }
        const saved = existing?.id
            ? await updateRoutingProfileMapping({ ...submitted, id: existing.id })
            : await createRoutingProfileMapping(submitted);
        mappings = [
            ...mappings.filter((mapping) => mapping.id !== saved.id),
            saved,
        ];
    }

    async function toggleAdvanced(mappingKey: string) {
        if (advancedMapping === mappingKey) {
            advancedMapping = "";
            return;
        }
        advancedMapping = mappingKey;
        await loadNativeControls(mappingKey);
    }

    async function loadNativeControls(mappingKey: string) {
        const selection = selections[mappingKey] ?? "";
        if (!selection) return;
        try {
            const input: Parameters<typeof routingNativeControls>[0] = {
                pluginId: plugin.id,
                instanceId: instance?.id,
                nativeConfig: cloneObject(nativeConfigs[mappingKey]),
            };
            if (selection.startsWith("profile:")) {
                input.profileId = selection.slice("profile:".length);
            } else {
                input.nativeProfileKey = selection.slice("native:".length);
            }
            const controls = await routingNativeControls(input);
            nativeControlsByMapping = {
                ...nativeControlsByMapping,
                [mappingKey]: controls,
            };
        } catch (error) {
            nativeControlsByMapping = {
                ...nativeControlsByMapping,
                [mappingKey]: {
                    pluginId: plugin.id,
                    instanceId: instance?.id,
                    groups: [],
                },
            };
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
        }
    }

    function controlValue(mappingKey: string, control: RoutingControl) {
        const nested = nestedValue(nativeConfigs[mappingKey], control.path ?? [control.key]);
        if (nested.found) return nested.value;
        return control.current ?? control.default;
    }

    function setControlValue(row: RoutingCategoryRow, control: RoutingControl, value: unknown) {
        const config = cloneObject(nativeConfigs[row.key]);
        setNestedValue(config, control.path ?? [control.key], value);
        const nextNativeConfigs = { ...nativeConfigs, [row.key]: config };
        for (const child of inheritedChildren(row)) {
            nextNativeConfigs[child.key] = cloneObject(config);
        }
        nativeConfigs = nextNativeConfigs;
        dirtyMappings = { ...dirtyMappings, [row.key]: true };
    }

    function inheritedChildren(row: RoutingCategoryRow) {
        if (row.subcategory) return [];
        return categoryRows.filter(
            (child) =>
                child.subcategory &&
                child.category.id === row.category.id &&
                !dirtyMappings[child.key] &&
                !editableMapping(child),
        );
    }

    function controlOptions(control: RoutingControl) {
        return (control.options ?? []).map((option) => ({
            text: metadataLabel(option.labels, option.label || option.value),
            value: option.value,
        }));
    }

    function groupLabel(group: RoutingNativeControls["groups"][number]) {
        return metadataLabel(group.labels, group.label || group.key);
    }

    function controlLabel(control: RoutingControl) {
        return metadataLabel(control.labels, control.label || control.key);
    }

    function metadataLabel(value: unknown, fallback: string) {
        return localizedText(objectValue(value) as Record<string, string>, $locale, fallback);
    }

    function controlNumber(mappingKey: string, control: RoutingControl) {
        const value = Number(controlValue(mappingKey, control));
        return Number.isFinite(value) ? value : Number(control.default ?? 0);
    }

    function handleUploadSelection(event: Event & { currentTarget: HTMLInputElement }) {
        const input = event.currentTarget;
        const file = input.files?.[0];
        input.value = "";
        if (file) void uploadProfile(file);
    }

    async function uploadProfile(file: File) {
        if (uploading) return;
        if (file.size > profileUploadMaxBytes) {
            show_toast({
                text: $_("routing-profile-file-too-large", {
                    values: { size: formatBytes(profileUploadMaxBytes) },
                }),
                icon: "close",
                type: "error",
            });
            return;
        }
        uploading = true;
        try {
            const name = file.name.replace(/\.[^.]+$/, "") || file.name;
            const saved = await createRoutingProfile({
                scope: "user",
                pluginId: plugin.id,
                key: uniqueProfileKey(name),
                name,
                kind: "custom_file",
                source: "user",
                contentBase64: arrayBufferToBase64(await file.arrayBuffer()),
                contentType: file.type || "text/plain",
                metadata: { filename: file.name },
                enabled: true,
            });
            profiles = [
                ...profiles.filter((profile) => profile.id !== saved.id),
                saved,
            ];
            show_toast({ text: $_("routing-profile-uploaded"), icon: "check", type: "success" });
            if (saved.mode === "other") {
                profileEditorModal?.openModal(saved);
            }
        } catch (error) {
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
        } finally {
            uploading = false;
        }
    }

    async function setProfileEnabled(profile: RoutingProfile, enabled: boolean) {
        if (!profile.id || profile.scope !== "user") return;
        try {
            const saved = await updateRoutingProfile({ ...profile, id: profile.id, enabled });
            profiles = profiles.map((candidate) => (candidate.id === saved.id ? saved : candidate));
        } catch (error) {
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
        }
    }

    function downloadProfile(profile: RoutingProfile) {
        try {
            const metadata = objectValue(profile.metadata);
            const filename = routingProfileDownloadFilename(
                metadata.filename,
                profile.name,
                profileFileExtension,
            );
            saveAs(
                new Blob([decodeRoutingProfileText(profile.contentBase64 ?? "")], {
                    type: profile.contentType || "text/plain;charset=utf-8",
                }),
                filename,
            );
        } catch (error) {
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
        }
    }

    function handleProfileSaved(saved: RoutingProfile) {
        profiles = profiles.map((profile) => (profile.id === saved.id ? saved : profile));
        nativeControlsByMapping = {};
    }

    function profileInUse(profile: RoutingProfile) {
        return Boolean(profile.id) && mappings.some((mapping) => mapping.profileId === profile.id);
    }

    function confirmProfileDeletion(profile: RoutingProfile) {
        if (profileInUse(profile)) return;
        profileToDelete = profile;
        profileDeleteModal?.openModal();
    }

    async function deleteProfile() {
        const profile = profileToDelete;
        profileToDelete = undefined;
        if (!profile?.id) return;
        try {
            await deleteRoutingProfile(profile.id);
            profiles = profiles.filter((candidate) => candidate.id !== profile.id);
            show_toast({ text: $_("routing-profile-deleted"), icon: "check", type: "success" });
        } catch (error) {
            show_toast({ text: errorMessage(error), icon: "close", type: "error" });
        }
    }

    function profileNativeConfig(profile: RoutingProfile | undefined) {
        if (!profile) return {};
        const metadata = objectValue(profile.metadata);
        return cloneObject(profile.nativeConfig ?? objectValue(metadata.nativeConfig));
    }

    function mappingKey(category: string, subcategory = "") {
        return `${category}\u0000${subcategory}`;
    }

    function domKey(key: string) {
        return key.replace(/[^a-zA-Z0-9_-]+/g, "-");
    }

    function objectValue(value: unknown): Record<string, unknown> {
        return value && typeof value === "object" && !Array.isArray(value)
            ? (value as Record<string, unknown>)
            : {};
    }

    function cloneObject(value: unknown): Record<string, unknown> {
        return cloneValue(objectValue(value)) as Record<string, unknown>;
    }

    function cloneValue(value: unknown): unknown {
        if (Array.isArray(value)) {
            return value.map(cloneValue);
        }
        if (value && typeof value === "object") {
            return Object.fromEntries(
                Object.entries(value as Record<string, unknown>).map(([key, item]) => [
                    key,
                    cloneValue(item),
                ]),
            );
        }
        return value;
    }

    function deepMerge(
        base: Record<string, unknown>,
        overlay: Record<string, unknown>,
    ): Record<string, unknown> {
        const result = cloneObject(base);
        for (const [key, value] of Object.entries(overlay)) {
            if (objectValue(value) === value && objectValue(result[key]) === result[key]) {
                result[key] = deepMerge(objectValue(result[key]), objectValue(value));
            } else {
                result[key] = cloneValue(value);
            }
        }
        return result;
    }

    function nestedValue(values: Record<string, unknown> | undefined, path: string[]) {
        let current: unknown = values;
        for (const key of path) {
            const object = objectValue(current);
            if (!Object.prototype.hasOwnProperty.call(object, key)) {
                return { found: false, value: undefined };
            }
            current = object[key];
        }
        return { found: true, value: current };
    }

    function setNestedValue(values: Record<string, unknown>, path: string[], value: unknown) {
        if (path.length === 0) return;
        let current = values;
        for (const key of path.slice(0, -1)) {
            const child = objectValue(current[key]);
            current[key] = { ...child };
            current = current[key] as Record<string, unknown>;
        }
        current[path.at(-1)!] = value;
    }

    function slug(value: string) {
        return (
            value
                .trim()
                .toLowerCase()
                .replace(/[^a-z0-9]+/g, "-")
                .replace(/^-+|-+$/g, "") || "custom-profile"
        );
    }

    function uniqueProfileKey(value: string) {
        const base = slug(value);
        const usedKeys = new Set(
            profiles
                .filter((profile) => profile.pluginId === plugin.id && profile.scope === "user")
                .map((profile) => profile.key),
        );
        let candidate = base.slice(0, 128);
        for (let suffix = 2; usedKeys.has(candidate); suffix += 1) {
            const ending = `-${suffix}`;
            candidate = `${base.slice(0, 128 - ending.length)}${ending}`;
        }
        return candidate;
    }

    function arrayBufferToBase64(buffer: ArrayBuffer) {
        const bytes = new Uint8Array(buffer);
        let binary = "";
        for (let offset = 0; offset < bytes.length; offset += 8192) {
            binary += String.fromCharCode(...bytes.subarray(offset, offset + 8192));
        }
        return btoa(binary);
    }

    function formatBytes(value: number) {
        return value >= 1024 ? `${Math.floor(value / 1024)} KB` : `${value} B`;
    }

    function errorMessage(error: unknown) {
        return error instanceof Error && error.message ? error.message : $_("error-generic");
    }

</script>

{#snippet mappingRow(
    row: RoutingCategoryRow,
    nested: boolean,
    expandable: boolean,
    expanded: boolean,
)}
    <div
        class="grid grid-cols-[minmax(0,1fr)_2.5rem] items-center gap-3 md:grid-cols-[minmax(13rem,0.85fr)_minmax(13rem,1.15fr)_2.5rem_2.5rem] {nested
            ? 'px-3 py-2'
            : 'p-3'}"
    >
        <div
            class="flex min-w-0 items-center gap-3 md:col-span-1"
            class:col-span-2={!expandable}
        >
            <span
                class="relative flex shrink-0 items-center justify-center bg-input-background text-content {nested
                    ? 'h-9 w-9 rounded-md text-lg'
                    : 'h-10 w-10 rounded-lg text-xl'}"
            >
                <i class="fa {categoryIcon(row)}"></i>
                {#if categoryBadgeIcon(row)}
                    <i
                        class="fa {categoryBadgeIcon(
                            row,
                        )} absolute right-0.5 top-0.5 text-[8px] text-gray-500"
                    ></i>
                {/if}
                {#if categoryBadge(row)}
                    <span
                        class="absolute -bottom-1 -right-1 max-w-10 truncate rounded-sm border border-input-border bg-background px-0.5 text-[7px] font-semibold leading-3 text-content"
                    >
                        {categoryBadge(row)}
                    </span>
                {/if}
            </span>
            <p class="min-w-0 truncate text-sm font-medium">{categoryLabel(row)}</p>
        </div>

        {#if expandable}
            <button
                type="button"
                class="col-start-2 row-start-1 flex h-10 w-10 items-center justify-center rounded-md text-content transition-colors hover:bg-input-background focus:outline-none focus:ring-2 focus:ring-input-ring md:col-start-4"
                aria-controls={`routing-category-${row.category.id}-subcategories`}
                aria-expanded={expanded}
                aria-label={expanded
                    ? $_("collapse-subcategories")
                    : $_("expand-subcategories")}
                title={expanded
                    ? $_("collapse-subcategories")
                    : $_("expand-subcategories")}
                onclick={() => toggleCategoryExpanded(row.category)}
            >
                <i class="fa {expanded ? 'fa-chevron-up' : 'fa-chevron-down'}"></i>
            </button>
        {/if}

        <div class="col-start-1 row-start-2 min-w-0 md:col-start-2 md:row-start-1">
            <SingleSelect
                ariaLabel={`${categoryLabel(row)}: ${$_("routing-native-profile")}`}
                items={profileItems()}
                value={selections[row.key] ?? ""}
                placeholder={$_("routing-select-profile")}
                onchange={(value) => selectProfile(row, value)}
            ></SingleSelect>
        </div>

        <button
            class="btn-secondary col-start-2 row-start-2 flex h-10 w-10 items-center justify-center !p-0 md:col-start-3 md:row-start-1"
            class:btn-disabled={!profileSupportsNativeControls(row.key)}
            class:bg-secondary-hover={advancedMapping === row.key}
            type="button"
            disabled={!profileSupportsNativeControls(row.key)}
            aria-controls={`routing-advanced-${domKey(row.key)}`}
            aria-label={`${categoryLabel(row)}: ${$_("routing-settings-advanced")}`}
            aria-expanded={advancedMapping === row.key}
            title={$_("routing-settings-advanced")}
            onclick={() => toggleAdvanced(row.key)}
        >
            <i class="fa fa-sliders"></i>
        </button>

        {#if advancedMapping === row.key}
            <div
                id={`routing-advanced-${domKey(row.key)}`}
                class="col-span-2 mt-1 space-y-4 border-t border-input-border pt-4 md:col-span-4"
            >
                {#if !nativeControlsByMapping[row.key]}
                    <div class="flex items-center gap-2 text-xs text-gray-500">
                        <span class="spinner inline-block h-4 w-4"></span>
                        <span>{$_("routing-native-controls-loading")}</span>
                    </div>
                {:else if nativeControlsByMapping[row.key]?.groups.length === 0}
                    <p class="text-xs text-gray-500">
                        {$_("routing-native-controls-empty")}
                    </p>
                {:else}
                    {#each nativeControlsByMapping[row.key]?.groups ?? [] as group (group.key)}
                        <div class="space-y-3">
                            <h5 class="text-sm font-medium">{groupLabel(group)}</h5>
                            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                                {#each group.controls as control (control.key)}
                                    <div>
                                        {#if control.type === "boolean" || control.valueType === "boolean"}
                                            <label class="inline-flex items-center gap-3 text-sm">
                                                <input
                                                    class="h-4 w-4"
                                                    type="checkbox"
                                                    checked={Boolean(controlValue(row.key, control))}
                                                    disabled={control.target !== "native_config"}
                                                    onchange={(event) =>
                                                        setControlValue(
                                                            row,
                                                            control,
                                                            event.currentTarget.checked,
                                                        )}
                                                />
                                                <span>{controlLabel(control)}</span>
                                            </label>
                                        {:else if control.options?.length}
                                            <SingleSelect
                                                label={controlLabel(control)}
                                                items={controlOptions(control)}
                                                value={controlValue(row.key, control)}
                                                disabled={control.target !== "native_config"}
                                                onchange={(value) =>
                                                    setControlValue(row, control, value)}
                                            ></SingleSelect>
                                        {:else if (control.type === "number" || control.valueType === "number") && control.min === undefined && control.max === undefined}
                                            <FiniteNumberField
                                                label={controlLabel(control)}
                                                extraClasses="!h-10 !px-3 !py-0"
                                                value={controlValue(row.key, control)}
                                                disabled={control.target !== "native_config"}
                                                onchange={(value) =>
                                                    setControlValue(
                                                        row,
                                                        control,
                                                        value,
                                                    )}
                                            ></FiniteNumberField>
                                        {:else if control.type === "number" || control.valueType === "number"}
                                            <label
                                                class="block text-sm font-medium"
                                                for={`routing-${domKey(row.key)}-${control.key}`}
                                            >
                                                {controlLabel(control)}
                                                <span class="ml-2 font-normal text-gray-500">
                                                    {controlNumber(row.key, control)}
                                                </span>
                                            </label>
                                            <input
                                                id={`routing-${domKey(row.key)}-${control.key}`}
                                                class="mt-2 w-full accent-toggle-active"
                                                type="range"
                                                min={control.min ?? 0}
                                                max={control.max ?? 1}
                                                step={control.step ?? 0.05}
                                                value={controlNumber(row.key, control)}
                                                disabled={control.target !== "native_config"}
                                                oninput={(event) =>
                                                    setControlValue(
                                                        row,
                                                        control,
                                                        Number(event.currentTarget.value),
                                                    )}
                                            />
                                        {:else}
                                            <TextField
                                                label={controlLabel(control)}
                                                extraClasses="!h-10 !px-3 !py-0"
                                                value={String(controlValue(row.key, control) ?? "")}
                                                disabled={control.target !== "native_config"}
                                                oninput={(event) =>
                                                    setControlValue(
                                                        row,
                                                        control,
                                                        event.currentTarget.value,
                                                    )}
                                            ></TextField>
                                        {/if}
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/each}
                {/if}
            </div>
        {/if}
    </div>
{/snippet}

<div class="space-y-6">
    {#if loading}
        <div class="flex items-center gap-3 py-6 text-sm text-gray-500">
            <span class="spinner inline-block h-5 w-5"></span>
            <span>{$_("routing-settings-loading")}</span>
        </div>
    {:else if loadError}
        <p class="rounded-md border border-red-400 p-3 text-sm text-red-400">{loadError}</p>
    {:else}
        <section class="space-y-3">
            <div>
                <h4 class="text-sm font-medium">{$_("routing-profile-mappings")}</h4>
                <p class="mt-1 text-xs text-gray-500">
                    {$_("routing-profile-mappings-help", { values: { plugin: plugin.name } })}
                </p>
            </div>
            <ol class="flex flex-col gap-3">
                {#each categoryGroups as group (group.category.id)}
                    {@const expanded = isCategoryExpanded(group.category)}
                    <li>
                        <div
                            class="border border-input-border transition-colors hover:bg-secondary-hover {expanded
                                ? 'rounded-t-lg rounded-bl-lg'
                                : 'rounded-lg'}"
                        >
                            {@render mappingRow(
                                group.row,
                                false,
                                group.children.length > 0,
                                expanded,
                            )}
                        </div>

                        {#if group.children.length > 0 && expanded}
                            <div
                                id={`routing-category-${group.category.id}-subcategories`}
                                class="-mt-px ml-6 rounded-b-lg border border-t-0 border-input-border bg-background md:ml-12"
                            >
                                <div class="divide-y divide-input-border">
                                    {#each group.children as child (child.key)}
                                        {@render mappingRow(child, true, false, false)}
                                    {/each}
                                </div>
                            </div>
                        {/if}
                    </li>
                {/each}
            </ol>
        </section>

        {#if profileUploadAvailable}
            <section class="space-y-3 border-t border-input-border pt-5">
                <div class="flex items-center justify-between gap-3">
                    <h4 class="text-sm font-medium">{$_("routing-custom-profiles")}</h4>
                    <input
                        class="hidden"
                        bind:this={profileUploadInput}
                        type="file"
                        accept={(uploadMetadata.extensions as string[] | undefined)?.join(",") ?? ""}
                        onchange={handleUploadSelection}
                    />
                    <button
                        class="btn-secondary flex h-10 w-10 items-center justify-center !p-0"
                        class:btn-disabled={uploading}
                        type="button"
                        disabled={uploading}
                        aria-label={$_("routing-upload-profile")}
                        title={$_("routing-upload-profile")}
                        onclick={() => profileUploadInput?.click()}
                    >
                        {#if uploading}
                            <span class="spinner inline-block h-4 w-4"></span>
                        {:else}
                            <i class="fa fa-plus"></i>
                        {/if}
                    </button>
                </div>

                {#if customProfiles.length > 0}
                    <div class="space-y-2">
                        {#each customProfiles as profile (profile.id)}
                            <div class="flex items-center justify-between gap-3 rounded-md border border-input-border px-3 py-2">
                                <div class="min-w-0">
                                    <p class="truncate text-sm font-medium">{profile.name}</p>
                                    {#if profile.kind === "custom_file" && profile.mode === "other"}
                                        <p class="mt-1 text-xs text-amber-400">
                                            <i class="fa fa-triangle-exclamation mr-1"></i>
                                            {$_("routing-profile-mode-unresolved")}
                                        </p>
                                    {/if}
                                </div>
                                <div class="flex shrink-0 items-center gap-2">
                                    {#if profile.kind === "custom_file"}
                                        <button
                                            class="btn-secondary flex h-10 w-10 items-center justify-center !p-0"
                                            type="button"
                                            aria-label={`${$_("edit")}: ${profile.name}`}
                                            title={$_("edit")}
                                            onclick={() => profileEditorModal?.openModal(profile)}
                                        >
                                            <i class="fa fa-pen"></i>
                                        </button>
                                        <button
                                            class="btn-secondary flex h-10 w-10 items-center justify-center !p-0"
                                            type="button"
                                            aria-label={`${$_("download")}: ${profile.name}`}
                                            title={$_("download")}
                                            onclick={() => downloadProfile(profile)}
                                        >
                                            <i class="fa fa-download"></i>
                                        </button>
                                    {/if}
                                    <button
                                        class="btn-secondary flex h-10 w-10 items-center justify-center !p-0"
                                        class:btn-disabled={profileInUse(profile)}
                                        type="button"
                                        disabled={profileInUse(profile)}
                                        aria-label={`${$_("delete")}: ${profile.name}`}
                                        title={profileInUse(profile)
                                            ? $_("routing-profile-delete-in-use")
                                            : $_("delete")}
                                        onclick={() => confirmProfileDeletion(profile)}
                                    >
                                        <i class="fa fa-trash"></i>
                                    </button>
                                    <Toggle
                                        name={`routing-profile-enabled-${profile.id}`}
                                        value={profile.enabled}
                                        disabled={profile.enabled && profileInUse(profile)}
                                        ariaLabel={`${profile.name}: ${profile.enabled
                                            ? $_("routing-profile-disable")
                                            : $_("routing-profile-enable")}`}
                                        onchange={(enabled) => setProfileEnabled(profile, enabled)}
                                    ></Toggle>
                                </div>
                            </div>
                        {/each}
                    </div>
                {/if}
            </section>
        {/if}
    {/if}
</div>

<RoutingProfileEditorModal
    id={profileEditorModalId}
    maxBytes={profileUploadMaxBytes}
    fileExtension={profileFileExtension}
    language={profileEditorLanguage}
    onsave={handleProfileSaved}
    bind:this={profileEditorModal}
></RoutingProfileEditorModal>

<ConfirmModal
    id={`routing-profile-delete-${domKey(plugin.id)}-${domKey(instance?.id ?? "default")}`}
    bind:this={profileDeleteModal}
    text={profileToDelete
        ? $_("routing-profile-delete-confirm", { values: { name: profileToDelete.name } })
        : ""}
    onconfirm={deleteProfile}
></ConfirmModal>
