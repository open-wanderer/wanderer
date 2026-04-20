<script lang="ts">
    import { goto, invalidateAll } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "$lib/components/base/button.svelte";
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import ApiTokenModal from "$lib/components/settings/api_token_modal.svelte";
    import ApiTokenSuccessModal from "$lib/components/settings/api_token_success_modal.svelte";
    import EmailModal from "$lib/components/settings/email_modal.svelte";
    import PasswordModal from "$lib/components/settings/password_modal.svelte";
    import type { APIToken } from "$lib/models/api_token";
    import {
        api_tokens_create,
        api_tokens_delete,
    } from "$lib/stores/api_token_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import {
        currentUser,
        logout,
        users_delete,
        users_update,
    } from "$lib/stores/user_store";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    const settings = page.data.settings;

    let selectedLanguage = "en";
    let selectedMapFocus = "trails";

    let citySearchQuery: string = "";

    let confirmModal: ConfirmModal;
    let emailModal: EmailModal;
    let passwordModal: PasswordModal;
    let tokenModal: ApiTokenModal;
    let tokenSuccessModal: ApiTokenSuccessModal;

    let tokenLoading: boolean = $state(false);
    let rawAPIToken: string | null = $state(null);

    onMount(() => {
        citySearchQuery = settings?.location?.name ?? "";
        selectedLanguage = settings?.language || "en";
        selectedMapFocus = settings?.mapFocus ?? "trails";
    });

    async function deleteAccount() {
        await users_delete($currentUser!);
        logout();
        goto("/");
    }

    async function updateEmail(email: string) {
        try {
            await users_update({ ...$currentUser!, email: email });
            show_toast({
                text: $_("email-updated"),
                icon: "check",
                type: "success",
            });
        } catch (e) {
            show_toast({
                text: "Error updating email",
                icon: "close",
                type: "error",
            });
        }
    }

    async function updatePassword(data: {
        oldPassword: string;
        password: string;
        passwordConfirm: string;
    }) {
        try {
            await users_update({ ...$currentUser!, ...data });
            show_toast({
                text: $_("password-updated"),
                icon: "check",
                type: "success",
            });
        } catch (e) {
            show_toast({
                text: $_("error-updating-password"),
                icon: "close",
                type: "error",
            });
        }
    }

    async function generateAPIToken(token: APIToken) {
        try {
            tokenLoading = true;
            const tokenResponse = await api_tokens_create(token);
            rawAPIToken = tokenResponse.rawToken;
            tokenSuccessModal.openModal();
            await invalidateAll();
        } catch (e) {
            show_toast({
                text: $_("error-generating-token"),
                icon: "close",
                type: "error",
            });
        } finally {
            tokenLoading = false;
        }
    }

    async function deleteAPIToken(token: APIToken) {
        try {
            const tokenResponse = await api_tokens_delete(token);
            await invalidateAll();
        } catch (e) {
            show_toast({
                text: $_("error-deleting-token"),
                icon: "close",
                type: "error",
            });
        }
    }
</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>
{#if $currentUser}
    <h2 class="text-2xl font-semibold">{$_("my-account")}</h2>
    <hr class="mt-4 mb-6 border-input-border" />
    <div class="space-y-6">
        <h4 class="text-xl font-medium">{$_("login-details")}</h4>
        <button
            class="btn-secondary block"
            onclick={() => emailModal.openModal()}>{$_("change-email")}</button
        >
        <button class="btn-secondary" onclick={() => passwordModal.openModal()}
            >{$_("change-password")}</button
        >
        <div>
            <div class="flex justify-between items-center">
                <h4 class="text-xl font-medium">{$_("api-tokens")}</h4>
                <Button
                    secondary
                    onclick={() => tokenModal.openModal()}
                    loading={tokenLoading}
                    ><i class="fa fa-plus mr-2"></i>
                    {$_("generate-new-token")}</Button
                >
            </div>
            <p class="mt-3">{$_("api-tokens-hint")}</p>
            {#if data.apiTokens.totalItems == 0}
                <p class="text-center pt-8 pb-2 text-gray-500">
                    {$_("no-api-tokens")}
                </p>
            {:else}
                <div
                    class="border border-input-border rounded-xl overflow-clip mt-4"
                >
                    <table class="api-token-table w-full table-auto">
                        <thead class="text-left">
                            <tr class="text-sm bg-secondary-hover">
                                <th>{$_("name")}</th>
                                <th>{$_("expiration")}</th>
                                <th>{$_("last-used")}</th>
                                <th></th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each data.apiTokens.items as token}
                                <tr class="border-t border-input-border">
                                    <td>{token.name}</td>
                                    <td
                                        >{token.expiration
                                            ? new Date(
                                                  token.expiration,
                                              ).toLocaleDateString(undefined, {
                                                  month: "long",
                                                  day: "2-digit",
                                                  year: "numeric",
                                                  timeZone: "UTC",
                                              })
                                            : $_("never")}</td
                                    >
                                    <td
                                        >{token.last_used
                                            ? new Date(
                                                  token.last_used,
                                              ).toLocaleTimeString(undefined, {
                                                  month: "2-digit",
                                                  day: "2-digit",
                                                  year: "numeric",
                                                  hour: "2-digit",
                                                  minute: "2-digit",
                                              })
                                            : "-"}</td
                                    >
                                    <td
                                        ><button
                                            onclick={() =>
                                                deleteAPIToken(token)}
                                            aria-label="delete api token"
                                            ><i class="fa fa-trash"></i></button
                                        ></td
                                    >
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
        <div class="space-y-4">
            <h4 class="text-xl text-red-400 font-medium">
                {$_("danger-zone")}
            </h4>
            <button
                id="delete-account"
                class="btn-danger"
                onclick={() => confirmModal.openModal()}
                >{$_("delete-account")}</button
            >
        </div>
    </div>
    <EmailModal
        email={$currentUser.email}
        onsave={updateEmail}
        bind:this={emailModal}
    ></EmailModal>
    <PasswordModal onsave={updatePassword} bind:this={passwordModal}
    ></PasswordModal>
    <ApiTokenModal onsave={generateAPIToken} bind:this={tokenModal}
    ></ApiTokenModal>
    <ApiTokenSuccessModal bind:token={rawAPIToken} bind:this={tokenSuccessModal}
    ></ApiTokenSuccessModal>
{/if}
<ConfirmModal
    text={$_("account-delete-confirm")}
    bind:this={confirmModal}
    onconfirm={deleteAccount}
></ConfirmModal>

<style>
    .api-token-table th,
    td {
        padding: 16px;
    }
</style>
