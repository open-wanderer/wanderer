<script lang="ts">
    import { afterNavigate, goto } from "$app/navigation";
    import { page } from "$app/state";
    import LogoText from "$lib/components/logo/logo_text.svelte";
    import { theme, toggleTheme } from "$lib/stores/theme_store";
    import { currentUser, logout } from "$lib/stores/user_store";
    import { getFileURL } from "$lib/util/file_util";
    import type { AuthRecord } from "pocketbase";
    import { _ } from "svelte-i18n";
    import { backInOut, cubicOut } from "svelte/easing";
    import { Tween } from "svelte/motion";
    import Drawer from "./base/drawer.svelte";
    import Dropdown from "./base/dropdown.svelte";
    import LogoTextLight from "./logo/logo_text_light.svelte";
    import NotificationDropdown from "./notification/notification_dropdown.svelte";
    import UrlImportModal from "./settings/url_import_modal.svelte";

    interface Props {
        user: AuthRecord;
    }

    let { user }: Props = $props();

    let navBarItems = [
        { text: "Home", value: "/" },
        { text: $_("trail", { values: { n: 2 } }), value: "/trails" },
        { text: $_("map"), value: "/map" },
        { text: $_("list", { values: { n: 2 } }), value: "/lists" },
    ];

    const dropdownItems = [
        { text: $_("profile"), value: "profile", icon: "user" },
        { text: $_("my-trails"), value: "trails", icon: "route" },

        { text: $_("settings"), value: "settings", icon: "cog" },
        { text: $_("logout"), value: "logout", icon: "right-from-bracket" },
    ];

    const importDropdownItems = [
        { text: $_("from-file"), value: "file", icon: "file-import" },
        { text: $_("from-url"), value: "url", icon: "server" },
    ];

    const indicatorPosition = new Tween(0, {
        duration: 300,
        easing: cubicOut,
    });

    const indicatorWidth = new Tween(0, {
        duration: 300,
        easing: cubicOut,
    });

    const indicatorScale = new Tween(0, {
        duration: 600,
        easing: backInOut,
    });

    let drawerOpen: boolean = $state(false);

    let urlImportModal: UrlImportModal;

    afterNavigate((e) => {
        const routeId = e.to?.route.id;
        const navBarLinks = document.getElementById("nav-bar-links");
        let childPosition = -1;
        switch (routeId) {
            case "/":
                childPosition = 1;
                break;
            case "/trails":
                childPosition = 2;
                break;
            case "/map":
                childPosition = 3;
                break;
            case "/lists/[[handle]]/[[id]]":
                childPosition = 4;
                break;
            default:
                break;
        }

        if (childPosition !== -1) {
            const childElement = navBarLinks?.children[
                childPosition
            ] as HTMLElement;
            const newWidth = childElement?.getBoundingClientRect().width ?? 0;
            const newPosition = childElement.offsetLeft;
            const padding = 16;
            indicatorScale.set(1);
            indicatorWidth.set(newWidth + padding);
            indicatorPosition.set(newPosition - padding / 2);
        } else {
            indicatorScale.set(0);
        }
    });

    function handleDropdownClick(item: { text: string; value: any }) {
        if (item.value == "profile") {
            goto(`/profile/@${$currentUser?.username?.toLowerCase()}`);
        } else if (item.value == "trails") {
            goto(`/profile/@${$currentUser?.username?.toLowerCase()}/trails`);
        } else if (item.value == "logout") {
            logout();
            window.location.href = "/";
        } else if (item.value == "settings") {
            goto("/settings/profile");
        }
    }

    function handleImportDropdownClick(item: { text: string; value: any }) {
        if (item.value == "file") {
            goto(`/settings/export`);
        } else if (item.value == "url") {
            urlImportModal.openModal();
        }
    }
</script>

<Drawer bind:open={drawerOpen}>
    <div class="flex gap-4 items-center m-4">
        <div class="basis-full"></div>
        <button
            aria-label="Toggle theme"
            class="btn-icon fa-regular fa-{$theme === 'light' ? 'sun' : 'moon'}"
            onclick={() => toggleTheme()}
        ></button>
        <button
            aria-label="Toggle drawer"
            class="btn-icon block fa fa-close float-right"
            onclick={() => (drawerOpen = false)}
        ></button>
    </div>
    <div class="flex flex-col px-12 gap-8">
        {#each navBarItems as item}
            <a class="font-semibold text-xl" href={item.value}>{item.text}</a>
        {/each}
    </div>
    <hr class="my-6 border-input-border" />
    <div class="flex flex-col basis-full">
        {#if user}
            <a class="btn-primary text-center mx-4" href="/trail/edit/new"
                ><i class="fa fa-plus mr-2"></i>{$_("new-trail")}</a
            >
            <div class="basis-full"></div>
            <hr class="border-input-border" />
            <div class="flex gap-4 items-center justify-between m-4">
                <a
                    class="shrink-0"
                    href="/profile/@{user.username.toLowerCase()}"
                >
                    <img
                        class="rounded-full w-10 aspect-square"
                        src={getFileURL(user, user.avatar) ||
                            `https://api.dicebear.com/7.x/initials/svg?seed=${user.username.toLowerCase()}&backgroundType=gradientLinear`}
                        alt="avatar"
                    />
                </a>
                <a
                    href="/profile/@{user.username.toLowerCase()}"
                    style="width: calc(100% - 104px)"
                >
                    <p class="text-sm overflow-hidden text-ellipsis">
                        {user.username}
                    </p>
                    <p
                        class="text-sm text-gray-500 overflow-hidden text-ellipsis"
                    >
                        {user.email}
                    </p>
                </a>
                <button
                    aria-label="Logout"
                    onclick={() => {
                        logout();
                        window.location.href = "/";
                    }}
                    class="btn-icon"
                    ><i class="fa-solid fa-arrow-right-from-bracket"
                    ></i></button
                >
            </div>
        {:else}
            <a class="btn-primary btn-large text-center mx-4" href="/login"
                >{$_("login")}</a
            >
        {/if}
    </div>
</Drawer>

<nav
    class="flex justify-between items-center p-6 {page.url.pathname === '/'
        ? 'sticky top-0 z-10 bg-background'
        : ''}"
>
    <a href="/">
        {#if $theme == "light"}
            <LogoText></LogoText>
        {:else}
            <LogoTextLight></LogoTextLight>
        {/if}
    </a>
    <menu id="nav-bar-links" class="hidden lg:flex gap-8 relative py-1 px-2">
        <div
            class="absolute h-full w-16 bg-menu-item-background-hover rounded-xl top-0 z-0"
            style="width: {indicatorWidth.current}px; left: {indicatorPosition.current}px; scale: {indicatorScale.current}"
        ></div>
        {#each navBarItems as item}
            <a class="font-semibold z-10" href={item.value}>{item.text}</a>
        {/each}
    </menu>
    {#if user}
        <div class="hidden lg:flex gap-6 items-center">
            <button
                aria-label="Toggle theme"
                class="btn-icon fa-regular fa-{$theme === 'light'
                    ? 'sun'
                    : 'moon'}"
                onclick={() => toggleTheme()}
            ></button>
            <nav data-sveltekit-reload>
                <div class="flex">
                    <a
                        class="btn-primary btn-large !rounded-r-none focus:ring-0"
                        href="/trail/edit/new"
                        ><i class="fa fa-plus mr-2"></i>{$_("new-trail")}</a
                    >
                    <Dropdown
                        items={importDropdownItems}
                        onchange={(item) => handleImportDropdownClick(item)}
                    >
                        {#snippet children({ toggleMenu: openDropdown })}
                            <button
                                onclick={openDropdown}
                                class="bg-primary rounded-r-lg text-white min-h-12 hover:bg-primary-hover px-3"
                                aria-label="Open trail create dropdown"
                                ><i class="fa fa-caret-down"></i></button
                            >
                        {/snippet}
                    </Dropdown>
                </div>
            </nav>
            {#if page.data.notifications}
                <NotificationDropdown></NotificationDropdown>
            {/if}
            <Dropdown
                items={dropdownItems}
                onchange={(item) => handleDropdownClick(item)}
            >
                {#snippet children({ toggleMenu: openDropdown })}
                    <div class="flex items-center">
                        <button
                            class="rounded-full bg-white text-black hover:bg-gray-200 focus:ring-4 ring-gray-100/50 transition-colors h-10 aspect-square"
                            onclick={openDropdown}
                        >
                            <img
                                class="rounded-full w-full h-full"
                                src={getFileURL(user, user.avatar) ||
                                    `https://api.dicebear.com/7.x/initials/svg?seed=${user.username.toLowerCase()}&backgroundType=gradientLinear`}
                                alt="avatar"
                            />
                        </button>
                    </div>
                {/snippet}
            </Dropdown>
        </div>
    {:else}
        <div class="hidden md:flex items-center gap-8">
            <button
                aria-label="Toggle theme"
                class="btn-icon fa-regular fa-{$theme === 'light'
                    ? 'sun'
                    : 'moon'}"
                onclick={() => toggleTheme()}
            ></button>
            <a class="btn-primary btn-large" href="/login">{$_("login")}</a>
        </div>
    {/if}
    <button
        aria-label="Toggle drawer"
        class="btn-icon fa fa-bars lg:hidden"
        onclick={() => (drawerOpen = !drawerOpen)}
    ></button>
</nav>

<UrlImportModal bind:this={urlImportModal}></UrlImportModal>
