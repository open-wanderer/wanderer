<script lang="ts">
    import Button from "$lib/components/base/button.svelte";
    import Editor from "$lib/components/base/editor.svelte";
    import type { SelectItem } from "$lib/components/base/select.svelte";
    import Select from "$lib/components/base/select.svelte";
    import type { Category } from "$lib/models/category.js";
    import { settings_update } from "$lib/stores/settings_store";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import { currentUser, users_update } from "$lib/stores/user_store";
    import { getFileURL } from "$lib/util/file_util";
    import SkillCard from "$lib/components/settings/skill_card.svelte";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    let settings = $state(data.settings)

    let selectedCategory = $state(
        data.categories.find((c) => c.id == data.settings.category) || data.categories[0],
    );

    let bio = $state(data.settings?.bio ?? "");

    const categoryItems: SelectItem[] = data.categories.map((c: Category) => ({
        text: $_(c.name),
        value: c,//.id,
    }));

    function openFileBrowser() {
        document.getElementById("avatarInput")!.click();
    }

    async function handleAvatarSelection() {
        if (!$currentUser) {
            return;
        }
        const files = (
            document.getElementById("avatarInput") as HTMLInputElement
        ).files;

        if (!files || files.length == 0) {
            return;
        }

        await users_update($currentUser!, files[0]);
    }

    async function handleBioSave() {
        if (!settings) {
            return;
        }
        try {
            settings.bio = bio;
            await settings_update(settings);
        } catch (e) {
            show_toast({
                type: "error",
                icon: "close",
                text: "Error saving bio",
            });
            console.error(e);
        }
    }

    async function handleCategorySelection(value: string) {
        await settings_update({
            id: settings.id,
            category: value,
        });
    }

    async function handleSkillUpdate() {
        await settings_update(settings);
    }
</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>
{#if $currentUser}
    <h2 class="text-2xl font-semibold">{$_("profile")}</h2>
    <hr class="mt-4 mb-6 border-input-border" />
    <div class="space-y-6">
        <div class="flex gap-6 items-center">
            <div
                class="rounded-full w-24 aspect-square overflow-hidden relative group"
            >
                <img
                    class="object-cover h-full"
                    src={getFileURL($currentUser, $currentUser.avatar) ||
                        `https://api.dicebear.com/7.x/initials/svg?seed=${$currentUser.username?.toLowerCase()}&backgroundType=gradientLinear`}
                    alt="avatar"
                />
                <button
                    aria-label="Open file browser"
                    class="absolute top-0 w-24 aspect-square opacity-0 group-hover:opacity-100 flex justify-center items-center bg-black/50 focus:bg-black/60 text-white cursor-pointer transition-opacity"
                    onclick={openFileBrowser}
                >
                    <i class="fa fa-pen"></i>
                </button>
                <input
                    type="file"
                    name="avatar"
                    id="avatarInput"
                    accept="image/*"
                    style="display: none;"
                    onchange={handleAvatarSelection}
                />
            </div>
            <div>
                <h4 class="text-xl font-semibold">{$currentUser.username}</h4>
                <h5 class="font-medium">{$currentUser.email}</h5>
            </div>
        </div>
        <div>
            <h4 class="text-xl font-medium">Bio</h4>
            <Editor bind:value={bio}></Editor>
            <div class="mt-3">
                <Button
                    onclick={() => handleBioSave()}
                    primary
                    disabled={settings.bio === bio}>{$_("save")}</Button
                >
            </div>
        </div>

        <div>
            <h4 class="text-xl font-medium mb-2">{$_("favourite-sport")}</h4>
            <Select
                items={categoryItems}
                bind:value={selectedCategory}
                onchange={handleCategorySelection}
            ></Select>
        </div>
        <div>
            <h4 class="text-xl font-medium mb-2">{$_("skills")}</h4>
            {#each categoryItems as category}
                <SkillCard
                    bind:settings={settings}
                    category={category.value}
                    onchange={handleSkillUpdate}
                />
            {/each}
        </div>
    </div>
{/if}
