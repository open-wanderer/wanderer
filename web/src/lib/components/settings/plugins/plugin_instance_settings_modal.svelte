<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import type { SelectItem } from "$lib/components/base/select.svelte";
    import SingleSelect from "$lib/components/base/single_select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import PluginAssetSettings from "$lib/components/settings/plugins/plugin_asset_settings.svelte";
    import PluginConfigFields from "$lib/components/settings/plugins/plugin_config_fields.svelte";
    import PluginMergeSettings from "$lib/components/settings/plugins/plugin_merge_settings.svelte";
    import PluginTrailSettings from "$lib/components/settings/plugins/plugin_trail_settings.svelte";
    import CategoryPicker from "$lib/components/trail/category_picker.svelte";
    import type { Category } from "$lib/models/category";
    import type { PluginInstance } from "$lib/models/plugin_instance";
    import type { ConfigField, PluginProvider } from "$lib/models/plugin_provider";
    import type { Subcategory } from "$lib/models/subcategory";
    import { plugin_auth_validate, plugin_oauth_start } from "$lib/stores/plugin_instance_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import {
        categoryMappingTargetFromUnknown,
        categoryMappingTargetToPickerValue,
        type CategoryMappingTarget,
    } from "$lib/util/category_util";
    import { translatePluginAPIError } from "$lib/util/plugin_error_i18n";
    import {
        configFieldDescription,
        configFieldLabel,
        configFieldOptionLabel,
        pluginTitle as localizedPluginTitle,
        providerCategoryLabel,
    } from "$lib/util/plugin_i18n";
    import { tick } from "svelte";
    import { _, locale } from "svelte-i18n";

    interface CategoryMappingRow {
        providerCategory: string;
        category: string;
    }

    type PluginInstanceForm = Partial<PluginInstance> & {
        mappingChanged?: boolean;
    };

    interface Props {
        plugin: PluginProvider;
        categories?: Category[];
        subcategories?: Subcategory[];
        instance?: PluginInstance;
        onbeforecategorymappingsave?: (instance: PluginInstanceForm) => Promise<boolean> | boolean;
        onsave?: (instance: Partial<PluginInstance>) => Promise<PluginInstance | void> | PluginInstance | void;
    }

    let { plugin, categories = [], subcategories = [], instance, onbeforecategorymappingsave, onsave }: Props = $props();

    let modal: Modal | undefined;
    let auth: Record<string, string> = $state(initialAuth());
    let planned = $state(true);
    let completed = $state(true);
    let mergeEnabled = $state(false);
    let privacy = $state("original");
    let photoMode = $state("copy");
    let maxPhotosPerTrail: string | number = $state("20");
    let maxPhotosPerWaypoint: string | number = $state("5");
    let autoAttachTrailPlugins = $state(true);
    let autoAttachUpload = $state(true);
    let categoryMappingRows: CategoryMappingRow[] = $state(initialCategoryMappingRows());
    let categoryMappingList: HTMLDivElement | undefined = $state();
    let authFields = $derived(plugin.auth.fields ?? []);
    let secretFields = $derived(new Set(plugin.auth.secretFields ?? plugin.auth.fields ?? []));
    let isOAuthPlugin = $derived(plugin.auth.type === "oauth2");
    let isSessionPlugin = $derived(plugin.auth.type === "session");
    let isConnected = $derived(instance?.status === "configured");
    let isSaving = $state(false);
    let needsOAuthConnect = $derived(isOAuthPlugin && (!isConnected || authChanged()));
    let plugin_id = $derived(plugin.id);
    let configSchema = $derived(plugin.configSchema ?? []);
    let serverUrlField = $derived(
        configSchema.find((field) => !field.hidden && field.key === "url"),
    );
    let extraConfig: Record<string, any> = $state(initialExtraConfig());
    let configErrors: Record<string, string> = $state({});
    let supportsPlanned = $derived(
        plugin.capabilities?.includes("list_routes.v1") ?? false,
    );
    let supportsCompleted = $derived(
        plugin.capabilities?.includes("list_activities.v1") ?? false,
    );
    let hasTourKindChoice = $derived(supportsPlanned && supportsCompleted);
    let supportsSourcePrivacy = $derived(
        plugin.capabilities?.includes("source_privacy") ?? false,
    );
    let supportsPhotoMode = $derived(plugin.type === "assets");
    let supportsPhotoLimits = $derived(plugin.type === "assets");
    let importSizeField = $derived(
        supportsPhotoMode
            ? configSchema.find((field) => !field.hidden && field.key === "importSize")
            : undefined,
    );
    let supportsMaxPhotosPerTrail = $derived(
        supportsPhotoLimits && hostConfigDeclares("maxPhotosPerTrail"),
    );
    let maxWaypointsField = $derived(
        supportsPhotoLimits
            ? configSchema.find((field) => !field.hidden && field.key === "maxWaypoints")
            : undefined,
    );
    let promotedConfigFieldKeys = $derived(
        new Set([serverUrlField?.key, importSizeField?.key, maxWaypointsField?.key]),
    );
    let visibleConfigSchema = $derived(
        configSchema.filter((field) => !field.hidden && !promotedConfigFieldKeys.has(field.key)),
    );
    let supportsAutoMerge = $derived(plugin.type === "trails");
    let mergeAvailable = $derived(
        supportsAutoMerge && (hostConfig().merge as any)?.available !== false,
    );
    let supportsCategoryMapping = $derived(supportsPlanned || supportsCompleted);
    let providerCategorySelectItems: SelectItem[] = $derived(providerCategoryItems());
    let canAddCategoryMappingRow = $derived(
        categoryMappingRows.every((row) => row.providerCategory && row.category) &&
        providerCategorySelectItems.some(
            (item) => !categoryMappingRows.some((row) => row.providerCategory === item.value),
        ),
    );

    const configLabels: Record<string, string> = {
        after: "ignore-trails-before-date",
    };
    const configHints: Record<string, string> = {
        after: "plugin-after-date-hint",
    };
    const privacySelectItems: SelectItem[] = [
        { text: $_("keep-original"), value: "original" },
        { text: $_("apply-user-settings"), value: "settings" },
    ];
    const photoModeSelectItems: SelectItem[] = [
        { text: $_("plugin-photo-mode-copy"), value: "copy" },
        { text: $_("plugin-photo-mode-link-private"), value: "link_private" },
    ];
    const defaultMaxPhotosPerTrail = 20;
    const defaultMaxPhotosPerWaypoint = 5;
    const authLabels: Record<string, string> = {
        email: $_("email"),
        password: $_("password"),
        clientId: "Client ID",
        clientSecret: "Client Secret",
    };

    function initialAuth() {
        return Object.fromEntries(
            (plugin.auth.fields ?? []).map((field) => [
                field,
                instance?.auth?.[field] == null ? "" : String(instance.auth[field]),
            ]),
        );
    }

    function initialExtraConfig(): Record<string, any> {
        const config = pluginConfig();
        return Object.fromEntries(
            configSchema.map((field) => {
                const saved = config[field.key];
                if (saved !== undefined) return [field.key, saved];
                if (field.type === "boolean") {
                    return [field.key, booleanDefault(field.default, false)];
                }
                if (field.default !== undefined && field.default !== null) {
                    if (field.type === "number") {
                        return [field.key, configNumberValue(field.default) ?? ""];
                    }
                    return [field.key, String(field.default)];
                }
                if (field.type === "select" && field.options?.length) {
                    return [field.key, field.options[0].value];
                }
                if (field.type === "select") {
                    return [field.key, ""];
                }
                if (field.type === "number" || field.type === "text" || field.type === "url") {
                    return [field.key, ""];
                }
                return [field.key, undefined];
            }),
        );
    }

    function initialCategoryMappingRows(): CategoryMappingRow[] {
        return Object.entries(categoryMapping())
            .filter(([, target]) => !isBlankCategoryMappingTarget(target))
            .map(([providerCategory, target]) => ({
                providerCategory,
                category: categoryPickerValue(target),
            }));
    }

    function manifestCategoryMapping(): Record<string, CategoryMappingTarget> {
        const raw = plugin.hostConfig?.categoryMapping;
        if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
            return {};
        }
        return categoryTargetMapping(raw as Record<string, unknown>);
    }

    function categoryMapping(): Record<string, CategoryMappingTarget> {
        const raw = hostConfig().categoryMapping;
        if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
            return {};
        }
        return categoryTargetMapping(raw as Record<string, unknown>);
    }

    function categoryMappingsEqual(
        left: Record<string, CategoryMappingTarget>,
        right: Record<string, CategoryMappingTarget>,
    ) {
        const leftKeys = Object.keys(left).sort();
        const rightKeys = Object.keys(right).sort();
        if (leftKeys.length !== rightKeys.length) {
            return false;
        }
        return leftKeys.every(
            (key, index) =>
                key === rightKeys[index] &&
                normalizedCategoryMappingTarget(left[key]) === normalizedCategoryMappingTarget(right[key]),
        );
    }

    function categoryTargetMapping(raw: Record<string, unknown>): Record<string, CategoryMappingTarget> {
        return Object.fromEntries(
            Object.entries(raw)
                .map(([key, value]) => [
                    key,
                    categoryMappingTargetFromUnknown(value),
                ])
                .filter(([, value]) => value !== undefined),
        );
    }

    function isBlankCategoryMappingTarget(target: CategoryMappingTarget): boolean {
        if (typeof target === "string") {
            return target.trim() === "";
        }
        return !target.category?.trim() && !target.subcategory?.trim();
    }

    function normalizedCategoryMappingTarget(target: CategoryMappingTarget): string {
        if (typeof target === "string") {
            return target.trim();
        }
        return JSON.stringify({
            category: target.category?.trim() ?? "",
            subcategory: target.subcategory?.trim() ?? "",
        });
    }

    function categoryPickerValue(target: CategoryMappingTarget): string {
        return categoryMappingTargetToPickerValue(
            target,
            categories,
            subcategories,
        );
    }

    function categoryMappingTargetFromPickerValue(value: string): CategoryMappingTarget {
        if (value.startsWith("subcategory:")) {
            const subcategoryId = value.replace("subcategory:", "");
            const subcategory = subcategories.find((candidate) => candidate.id === subcategoryId);
            return {
                category: subcategory?.category ?? "",
                subcategory: subcategoryId,
            };
        }
        if (value.startsWith("category:")) {
            return {
                category: value.replace("category:", ""),
                subcategory: "",
            };
        }
        return "";
    }

    function categoryPickerCurrentCategoryId(value: string): string | null {
        if (value.startsWith("category:")) {
            return value.replace("category:", "");
        }
        if (value.startsWith("subcategory:")) {
            const subcategoryId = value.replace("subcategory:", "");
            return subcategories.find((candidate) => candidate.id === subcategoryId)?.category ?? null;
        }
        return null;
    }

    function providerCategoryItems(): SelectItem[] {
        const values = new Set<string>([
            ...Object.keys(manifestCategoryMapping()),
            ...Object.keys(categoryMapping()),
            ...categoryMappingRows.map((row) => row.providerCategory).filter(Boolean),
        ]);
        return [...values]
            .map((value) => ({
                text: providerCategoryLabel(plugin, value, $locale),
                value,
            }))
            .sort((a, b) => a.text.localeCompare(b.text, $locale ?? undefined));
    }

    function providerCategoryItemsForRow(index: number): SelectItem[] {
        const currentValue = categoryMappingRows[index]?.providerCategory;
        const assignedValues = new Set(
            categoryMappingRows
                .filter((_, i) => i !== index)
                .map((row) => row.providerCategory)
                .filter(Boolean),
        );
        return providerCategorySelectItems.filter(
            (item) => item.value === currentValue || !assignedValues.has(item.value),
        );
    }

    function booleanDefault(value: unknown, fallback: boolean): boolean {
        if (typeof value === "boolean") return value;
        if (typeof value === "string") return value.trim().toLowerCase() === "true";
        return fallback;
    }

    function selectItems(field: ConfigField): SelectItem[] {
        return (field.options ?? []).map((o) => ({
            text: configFieldOptionLabel(o, $locale, $_(o.value)),
            value: o.value,
        }));
    }

    function fieldLabel(field: ConfigField): string {
        return configFieldLabel(field, $locale, $_(configLabels[field.key] ?? field.key));
    }

    function fieldHint(field: ConfigField): string | undefined {
        const description = configFieldDescription(field, $locale);
        if (description) return description;
        const hint = configHints[field.key];
        return hint ? $_(hint) : undefined;
    }

    function fieldError(field: ConfigField): string {
        return configErrors[field.key] ?? "";
    }

    async function addCategoryMappingRow() {
        const newIndex = categoryMappingRows.length;
        categoryMappingRows = [
            ...categoryMappingRows,
            {
                providerCategory: "",
                category: "",
            },
        ];
        await tick();
        categoryMappingList?.querySelector<HTMLElement>(`[data-category-mapping-row="${newIndex}"]`)
            ?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }

    function removeCategoryMappingRow(index: number) {
        categoryMappingRows = categoryMappingRows.filter((_, i) => i !== index);
    }

    function validateConfig(): boolean {
        const errors: Record<string, string> = {};
        const hiddenMissing: ConfigField[] = [];
        for (const field of configSchema) {
            const value = extraConfig[field.key];
            if (value === undefined || value === null || value === "") {
                if (field.required && field.hidden) {
                    hiddenMissing.push(field);
                } else if (field.required) {
                    errors[field.key] = $_("required");
                }
                continue;
            }
            if (field.type === "number") {
                const parsed = configNumberValue(value);
                if (parsed === undefined) {
                    errors[field.key] = $_("plugin-number-invalid");
                    continue;
                }
                const min = configNumberValue(field.min);
                if (min !== undefined && parsed < min) {
                    errors[field.key] = $_("plugin-number-min", { values: { min } });
                    continue;
                }
                const max = configNumberValue(field.max);
                if (max !== undefined && parsed > max) {
                    errors[field.key] = $_("plugin-number-max", { values: { max } });
                }
            }
        }
        configErrors = errors;
        if (hiddenMissing.length > 0) {
            show_toast({
                text: `${hiddenMissing.map((field) => fieldLabel(field)).join(", ")}: ${$_("required")}`,
                icon: "close",
                type: "error",
            });
        }
        return Object.keys(errors).length === 0 && hiddenMissing.length === 0;
    }

    function configNumberValue(value: unknown): number | undefined {
        if (value === undefined || value === null || value === "") {
            return undefined;
        }
        const parsed = Number(value);
        return Number.isFinite(parsed) ? parsed : undefined;
    }

    function authLabel(field: string): string {
        return authLabels[field] ?? field;
    }

    function authFieldType(field: string): "password" | "text" {
        return secretFields.has(field) ? "password" : "text";
    }

    function configSection(key: string): Record<string, any> {
        const section = instance?.config?.[key];
        if (section && typeof section === "object" && !Array.isArray(section)) {
            return section as Record<string, any>;
        }
        return {};
    }

    function pluginConfig(): Record<string, any> {
        return configSection("plugin");
    }

    function hostConfig(): Record<string, any> {
        const manifestConfig = plugin.hostConfig ?? {};
        const instanceConfig = configSection("host");
        const manifestMerge =
            manifestConfig.merge && typeof manifestConfig.merge === "object"
                ? (manifestConfig.merge as Record<string, unknown>)
                : {};
        const instanceMerge =
            instanceConfig.merge && typeof instanceConfig.merge === "object"
                ? (instanceConfig.merge as Record<string, unknown>)
                : {};
        const manifestAutoAttach =
            manifestConfig.autoAttach && typeof manifestConfig.autoAttach === "object"
                ? (manifestConfig.autoAttach as Record<string, unknown>)
                : {};
        const instanceAutoAttach =
            instanceConfig.autoAttach && typeof instanceConfig.autoAttach === "object"
                ? (instanceConfig.autoAttach as Record<string, unknown>)
                : {};
        return {
            ...manifestConfig,
            ...instanceConfig,
            merge: {
                ...manifestMerge,
                ...instanceMerge,
            },
            autoAttach: {
                ...manifestAutoAttach,
                ...instanceAutoAttach,
            },
        };
    }

    function instanceHostConfigOverrides(): Record<string, unknown> {
        const source = hostConfig();
        const projected: Record<string, unknown> = {};
        for (const key of [
            "planned",
            "completed",
            "privacy",
            "createSummitLogForCompleted",
            "categoryMapping",
            "categoryMappingUpdatedAt",
            "photoMode",
            "maxPhotosPerTrail",
            "maxPhotosPerWaypoint",
            "maxPhotosPerSummitLog",
        ]) {
            if (Object.prototype.hasOwnProperty.call(source, key)) {
                projected[key] = source[key];
            }
        }
        const merge = source.merge;
        if (merge && typeof merge === "object" && Object.prototype.hasOwnProperty.call(merge, "enabled")) {
            projected.merge = { enabled: (merge as Record<string, unknown>).enabled };
        }
        const autoAttach = source.autoAttach;
        if (autoAttach && typeof autoAttach === "object") {
            const projectedAutoAttach: Record<string, unknown> = {};
            for (const key of ["trailPlugins", "upload"]) {
                if (Object.prototype.hasOwnProperty.call(autoAttach, key)) {
                    projectedAutoAttach[key] = (autoAttach as Record<string, unknown>)[key];
                }
            }
            if (Object.keys(projectedAutoAttach).length > 0) {
                projected.autoAttach = projectedAutoAttach;
            }
        }
        return projected;
    }

    function hostConfigDeclares(key: string): boolean {
        return Object.prototype.hasOwnProperty.call(plugin.hostConfig ?? {}, key);
    }

    function authChanged() {
        for (const field of authFields) {
            const value = auth[field] ?? "";
            if (secretFields.has(field)) {
                if (value !== "") {
                    return true;
                }
                continue;
            }
            if (value !== ((instance?.auth?.[field] as string | undefined) ?? "")) {
                return true;
            }
        }
        return false;
    }

    export function openModal() {
        auth = initialAuth();
        const config = hostConfig();
        planned = supportsPlanned && (!hasTourKindChoice || ((config.planned as boolean | undefined) ?? true));
        completed =
            supportsCompleted && (!hasTourKindChoice || ((config.completed as boolean | undefined) ?? true));
        mergeEnabled = mergeAvailable && Boolean((config.merge as any)?.enabled);
        privacy = (config.privacy as string | undefined) ?? "original";
        photoMode = normalizedPhotoMode(config.photoMode);
        if (supportsMaxPhotosPerTrail) {
            maxPhotosPerTrail = photoLimitInputValue(config.maxPhotosPerTrail, defaultMaxPhotosPerTrail);
        }
        maxPhotosPerWaypoint = photoLimitInputValue(config.maxPhotosPerWaypoint, defaultMaxPhotosPerWaypoint);
        const autoAttach = normalizedAutoAttachConfig(config);
        autoAttachTrailPlugins = autoAttach.trailPlugins;
        autoAttachUpload = autoAttach.upload;
        extraConfig = initialExtraConfig();
        categoryMappingRows = initialCategoryMappingRows();
        configErrors = {};
        modal?.openModal();
    }

    function closeSettingsModal() {
        modal?.closeModal();
    }

    function pluginInstanceFromForm(): PluginInstanceForm {
        const submittedAuth: Record<string, string> = {};
        for (const field of authFields) {
            submittedAuth[field] = auth[field] ?? "";
        }

        const pluginRuntimeConfig: Record<string, unknown> = { ...pluginConfig() };
        const pluginHostConfig = instanceHostConfigOverrides();
        if (supportsPlanned) {
            pluginHostConfig.planned = hasTourKindChoice ? planned : true;
        } else {
            delete pluginHostConfig.planned;
        }
        if (supportsCompleted) {
            pluginHostConfig.completed = hasTourKindChoice ? completed : true;
        } else {
            delete pluginHostConfig.completed;
        }
        if (supportsSourcePrivacy) {
            pluginHostConfig.privacy = privacy;
        }
        if (supportsPhotoMode) {
            pluginHostConfig.photoMode = photoMode;
            pluginHostConfig.autoAttach = {
                trailPlugins: autoAttachTrailPlugins,
                upload: autoAttachUpload,
            };
            delete pluginHostConfig.providers;
        }
        if (supportsPhotoLimits) {
            if (supportsMaxPhotosPerTrail) {
                pluginHostConfig.maxPhotosPerTrail = normalizedPhotoLimit(maxPhotosPerTrail, defaultMaxPhotosPerTrail);
            } else {
                delete pluginHostConfig.maxPhotosPerTrail;
            }
            pluginHostConfig.maxPhotosPerWaypoint = normalizedPhotoLimit(maxPhotosPerWaypoint, defaultMaxPhotosPerWaypoint);
        } else {
            delete pluginHostConfig.maxPhotosPerTrail;
            delete pluginHostConfig.maxPhotosPerWaypoint;
        }
        if (supportsAutoMerge) {
            const currentMergeConfig =
                pluginHostConfig.merge && typeof pluginHostConfig.merge === "object"
                    ? (pluginHostConfig.merge as Record<string, unknown>)
                    : {};
            pluginHostConfig.merge = {
                ...currentMergeConfig,
                enabled: mergeAvailable && mergeEnabled,
            };
        } else {
            delete pluginHostConfig.merge;
        }
        const categoryMappingConfig: Record<string, CategoryMappingTarget> = {};
        const assignedProviderCategories = new Set<string>();
        for (const row of categoryMappingRows) {
            const providerCategory = row.providerCategory.trim();
            if (!providerCategory || !row.category) {
                continue;
            }
            assignedProviderCategories.add(providerCategory);
            categoryMappingConfig[providerCategory] = categoryMappingTargetFromPickerValue(row.category);
        }
        for (const providerCategory of [
            ...Object.keys(manifestCategoryMapping()),
            ...Object.keys(categoryMapping()),
        ]) {
            if (!assignedProviderCategories.has(providerCategory)) {
                categoryMappingConfig[providerCategory] = "";
            }
        }
        const mappingChanged = !categoryMappingsEqual(categoryMapping(), categoryMappingConfig);
        if (mappingChanged) {
            pluginHostConfig.categoryMappingUpdatedAt = new Date().toISOString();
        }
        pluginHostConfig.categoryMapping = categoryMappingConfig;
        for (const field of configSchema) {
            const val = extraConfig[field.key];
            if (val !== undefined && val !== "") {
                pluginRuntimeConfig[field.key] = field.type === "number" ? configNumberValue(val) : val;
            } else {
                delete pluginRuntimeConfig[field.key];
            }
        }
        const config: Record<string, unknown> = {
            plugin: pluginRuntimeConfig,
            host: pluginHostConfig,
        };

        const status = isOAuthPlugin
            ? !instance?.id
                ? "needs_auth"
                : authChanged()
                  ? isConnected
                      ? "needs_reauth"
                      : "needs_auth"
                  : instance.status
            : instance?.enabled
              ? "configured"
              : "disabled";

        return {
            ...instance,
            plugin_id,
            enabled: isOAuthPlugin && status !== "configured" ? false : (instance?.enabled ?? false),
            auth: submittedAuth,
            config,
            status,
            mappingChanged,
        };
    }

    async function validateAuthIfNeeded() {
        if (!isSessionPlugin || !authChanged()) {
            return;
        }
        await plugin_auth_validate({
            pluginId: plugin.id,
            instanceId: instance?.id,
            auth,
        });
    }

    async function submit() {
        if (!validateConfig()) {
            return;
        }
        try {
            isSaving = true;
            await validateAuthIfNeeded();
        } catch (e) {
            show_toast({
                text: translatePluginAPIError(e, $_("error-setting-up-plugin", { values: { provider: pluginTitle() } })),
                icon: "close",
                type: "error",
            });
            isSaving = false;
            return;
        }

        try {
            let candidate = pluginInstanceFromForm();
            if (supportsPhotoMode) {
                candidate = await applyAssetPluginCheckResult(candidate);
            }
            if (candidate.mappingChanged) {
                const shouldSave = await onbeforecategorymappingsave?.(candidate);
                if (shouldSave === false) {
                    return;
                }
                delete candidate.mappingChanged;
            }
            await onsave?.(candidate as Partial<PluginInstance>);
            closeSettingsModal();
        } catch (e) {
            if (supportsPhotoMode && e instanceof Error) {
                show_toast({
                    text: e.message,
                    icon: "close",
                    type: "error",
                });
            }
        } finally {
            isSaving = false;
        }
    }

    function pluginTitle() {
        return localizedPluginTitle(plugin, $locale);
    }

    async function startOAuth() {
        if (!validateConfig()) {
            return;
        }
        try {
            const candidate = pluginInstanceFromForm();
            delete candidate.mappingChanged;
            const saved = await onsave?.(candidate);
            const instanceId = saved?.id ?? instance?.id;
            if (!instanceId) {
                return;
            }
            const redirectUri = `${window.location.origin}/settings/plugins/oauth/callback`;
            const result = await plugin_oauth_start({
                pluginId: plugin.id,
                instanceId,
                redirectUri,
            });
            sessionStorage.setItem(`wanderer_plugin_oauth_${result.state}`, result.instanceId);
            window.location.href = result.url;
        } catch (e) {
            show_toast({
                text: e instanceof Error ? e.message : $_("error-starting-oauth"),
                icon: "close",
                type: "error",
            });
        }
    }

    function normalizedPhotoMode(value: unknown): string {
        return value === "link_private" ? value : "copy";
    }

    function normalizedPhotoLimit(value: unknown, fallback: number): number {
        const parsed = Number(value);
        if (!Number.isFinite(parsed) || parsed <= 0) {
            return fallback;
        }
        return Math.floor(parsed);
    }

    function photoLimitInputValue(value: unknown, fallback: number): string {
        return String(normalizedPhotoLimit(value, fallback));
    }

    function normalizedAutoAttachConfig(config: Record<string, any>): {
        trailPlugins: boolean;
        upload: boolean;
    } {
        if (Array.isArray(config.providers)) {
            const providers = config.providers.filter((value): value is string => typeof value === "string");
            return {
                trailPlugins: providers.some((provider) => provider !== "upload"),
                upload: providers.includes("upload"),
            };
        }
        const raw = config.autoAttach;
        if (raw && typeof raw === "object" && !Array.isArray(raw)) {
            return {
                trailPlugins: (raw as Record<string, unknown>).trailPlugins !== false,
                upload: (raw as Record<string, unknown>).upload !== false,
            };
        }
        return { trailPlugins: true, upload: true };
    }

    async function applyAssetPluginCheckResult(instance: PluginInstanceForm): Promise<PluginInstanceForm> {
        const response = await fetch(`/api/v1/plugins/assets/${encodeURIComponent(plugin.id)}/check`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                auth: instance.auth ?? {},
                config: instance.config ?? {},
            }),
        });
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.message ?? $_("error-setting-up-plugin", { values: { provider: pluginTitle() } }));
        }
        const result = (await response.json()) as { userId?: string };
        if (!result.userId) {
            return instance;
        }
        const config = {
            ...instance.config,
            plugin: {
                ...((instance.config?.plugin as Record<string, unknown> | undefined) ?? {}),
                userId: result.userId,
            },
        };
        return { ...instance, config };
    }
</script>

<Modal
    id="{plugin_id}-plugin-settings-modal"
    size="md:min-w-xl lg:min-w-2xl"
    title={pluginTitle() + " " + $_("settings")}
    bind:this={modal}
>
    {#snippet content()}
        <form
            id="{plugin_id}-plugin-settings-form"
            class="space-y-3"
            onsubmit={(event) => {
                event.preventDefault();
                submit();
            }}
        >
            <PluginConfigFields
                fields={serverUrlField ? [serverUrlField] : []}
                bind:config={extraConfig}
                {fieldLabel}
                {fieldHint}
                {fieldError}
                {selectItems}
                bordered={false}
                columns={1}
            />

            {#each authFields as field}
                <TextField
                    label={authLabel(field)}
                    placeholder={secretFields.has(field)
                        ? instance?.id
                            ? `(${$_("unchanged")})`
                            : ""
                        : field == "email"
                          ? "user@example.com"
                          : ""}
                    bind:value={auth[field]}
                    name={field}
                    type={authFieldType(field)}
                ></TextField>
            {/each}

            {#if isOAuthPlugin}
                <p class="text-xs text-gray-500 max-w-lg">
                    {#if isConnected}
                        {$_("plugin-oauth-connected-hint")}
                    {:else}
                        {$_("plugin-oauth-needs-connect-hint")}
                    {/if}
                </p>
            {/if}

            <PluginTrailSettings
                {hasTourKindChoice}
                {supportsSourcePrivacy}
                bind:planned={planned}
                bind:completed={completed}
                bind:privacy={privacy}
                {privacySelectItems}
            />

            {#if supportsPhotoMode}
                <PluginAssetSettings
                    bind:photoMode={photoMode}
                    {photoModeSelectItems}
                    {importSizeField}
                    {maxWaypointsField}
                    {supportsPhotoLimits}
                    {supportsMaxPhotosPerTrail}
                    bind:maxPhotosPerTrail={maxPhotosPerTrail}
                    bind:maxPhotosPerWaypoint={maxPhotosPerWaypoint}
                    bind:autoAttachTrailPlugins={autoAttachTrailPlugins}
                    bind:autoAttachUpload={autoAttachUpload}
                    bind:config={extraConfig}
                    {fieldLabel}
                    {fieldError}
                    {selectItems}
                />
            {/if}

            <PluginConfigFields
                fields={visibleConfigSchema}
                bind:config={extraConfig}
                {fieldLabel}
                {fieldHint}
                {fieldError}
                {selectItems}
            />

            {#if supportsCategoryMapping && categories.length > 0}
                <div class="space-y-2 pt-4 border-t border-input-border">
                    <div class="flex items-center justify-between gap-3">
                        <div>
                            <h4 class="text-sm font-medium">{$_("category-mapping")}</h4>
                            <p class="text-xs text-gray-500 mt-1">
                                {$_("category-mapping-help")}
                            </p>
                        </div>
                        <button
                            class="btn-primary text-sm"
                            class:btn-disabled={!canAddCategoryMappingRow}
                            type="button"
                            disabled={!canAddCategoryMappingRow}
                            onclick={addCategoryMappingRow}
                        >{$_("add-entry")}</button>
                    </div>
                    {#if categoryMappingRows.length > 0}
                        <div class="hidden md:grid grid-cols-[minmax(0,1.2fr)_minmax(14rem,1fr)_2.75rem] gap-3 text-sm font-medium">
                            <span>{$_("provider-category")}</span>
                            <span>{$_("category")} / {$_("subcategory")}</span>
                            <span></span>
                        </div>
                        <div
                            bind:this={categoryMappingList}
                            class="space-y-3 pr-1 scroll-smooth"
                            class:max-h-[300px]={categoryMappingRows.length > 6}
                            class:overflow-y-auto={categoryMappingRows.length > 6}
                            class:overflow-y-visible={categoryMappingRows.length <= 6}
                        >
                            {#each categoryMappingRows as row, i}
                                {@const providerItems = providerCategoryItemsForRow(i)}
                                <div
                                    class="grid grid-cols-1 md:grid-cols-[minmax(0,1.2fr)_minmax(14rem,1fr)_2.75rem] items-end gap-3"
                                    data-category-mapping-row={i}
                                >
                                    <SingleSelect
                                        ariaLabel={$_("provider-category")}
                                        placeholder={$_("select-provider-category")}
                                        items={providerItems}
                                        bind:value={row.providerCategory}
                                        disabled={row.providerCategory !== "" && providerItems.length <= 1}
                                    ></SingleSelect>
                                    <CategoryPicker
                                        value={row.category}
                                        label=""
                                        placeholder={$_("select-category")}
                                        currentCategoryId={categoryPickerCurrentCategoryId(row.category)}
                                        fixedDropdown
                                        onchange={(selection) => {
                                            row.category = selection.subcategory
                                                ? `subcategory:${selection.subcategory}`
                                                : `category:${selection.category}`;
                                        }}
                                    ></CategoryPicker>
                                    <button
                                        class="btn-icon h-10"
                                        type="button"
                                        onclick={() => removeCategoryMappingRow(i)}
                                        aria-label={$_("remove")}
                                    ><i class="fa fa-close"></i></button>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            {/if}

            {#if mergeAvailable}
                <PluginMergeSettings prefix="merge" bind:value={mergeEnabled} />
            {/if}
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={closeSettingsModal} disabled={isSaving}
                >{$_("cancel")}</button
            >
            {#if isOAuthPlugin}
                {#if needsOAuthConnect}
                    <button class="btn-primary" type="button" onclick={startOAuth} disabled={isSaving}
                        >{isConnected ? $_("save-and-reconnect") : $_("save-and-connect")}</button
                    >
                {:else}
                    <button
                        class="btn-primary"
                        form="{plugin_id}-plugin-settings-form"
                        type="submit"
                        name="save"
                        disabled={isSaving}>{$_("save")}</button
                    >
                {/if}
            {:else}
                <button
                    class="btn-primary"
                    form="{plugin_id}-plugin-settings-form"
                    type="submit"
                    name="save"
                    disabled={isSaving}
                    >{isSessionPlugin && authChanged() ? $_("save-and-validate") : $_("save")}</button
                >
            {/if}
        </div>
    {/snippet}
</Modal>
