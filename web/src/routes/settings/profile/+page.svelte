<script lang="ts">
    import Button from "$lib/components/base/button.svelte";
    import Editor from "$lib/components/base/editor.svelte";
    import { settings_update } from "$lib/stores/settings_store";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import { currentUser, users_update } from "$lib/stores/user_store";
    import {
        designSelectableCategories,
        displayCategoryIcon,
        displayCategoryName,
    } from "$lib/util/category_util";
    import { getFileURL } from "$lib/util/file_util";
    import { untrack } from "svelte";
    import { _, locale } from "svelte-i18n";

    let { data } = $props();

    let settings = $state(untrack(() => data.settings));

    let bio = $state(untrack(() => data.settings?.bio ?? ""));

    // Read-only display of the user's most important category; managed via priorities.
    let favouriteCategory = $derived(
        designSelectableCategories(
            data.categories,
            data.categoryPreferences,
            $locale,
        )[0],
    );

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
            <div class="flex items-center gap-3">
                {#if favouriteCategory}
                    <span class="flex items-center gap-2">
                        <i class="fa {displayCategoryIcon(favouriteCategory)}"></i>
                        {displayCategoryName(favouriteCategory, $locale)}
                    </span>
                {:else}
                    <span class="text-gray-500">—</span>
                {/if}
                <a
                    href="/settings/categories"
                    class="btn-icon tooltip inline-flex items-center justify-center"
                    aria-label={$_("edit")}
                    data-title={$_("edit")}
                >
                    <i class="fa fa-pen text-sm"></i>
                </a>
            </div>
        </div>
    </div>
{/if}
