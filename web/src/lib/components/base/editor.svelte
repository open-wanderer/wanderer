<script lang="ts">
    import { Editor, mergeAttributes } from "@tiptap/core";
    import { Link } from "@tiptap/extension-link";
    import Mention from "@tiptap/extension-mention";
    import Placeholder from "@tiptap/extension-placeholder";
    import { Underline } from "@tiptap/extension-underline";
    import StarterKit from "@tiptap/starter-kit";
    import { mount, onDestroy, onMount, unmount, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import { z, ZodError } from "zod";
    import { type DropdownItem } from "./dropdown.svelte";
    import Modal from "./modal.svelte";
    import {
        default as DropdownList,
        default as SearchList,
    } from "./search_list.svelte";
    import { type SelectItem } from "./select.svelte";
    import TextField from "./text_field.svelte";
    import Toggle from "./toggle.svelte";
    import type { SearchItem } from "./search.svelte";
    import type { SuggestionProps } from "@tiptap/suggestion";
    import type { Actor } from "$lib/models/activitypub/actor";
    import { searchActors } from "$lib/stores/search_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";

    let element: HTMLElement;
    let editor: Editor | undefined = $state();

    let modal: Modal;

    interface Props {
        value?: string;
        label?: string;
        error?: string | string[] | null;
        placeholder?: string;
        extraClasses?: string;
        searchListPosition?: string;
    }

    let {
        value = $bindable(""),
        label = "",
        error = "",
        placeholder = "",
        extraClasses = "",
        searchListPosition = "absolute"
    }: Props = $props();

    const fontSizes: SelectItem[] = [
        { text: $_("paragraph"), value: "p" },
        { text: `${$_("heading")} 1`, value: "h1" },
        { text: `${$_("heading")} 2`, value: "h2" },
        { text: `${$_("heading")} 3`, value: "h3" },
        { text: `${$_("heading")} 4`, value: "h4" },
        { text: `${$_("heading")} 5`, value: "h5" },
    ];

    let currentFontSize: string = $state("p");
    let linkURL: string = $state("");
    let linkURLError: string = $state("");
    let linkText: string = $state("");
    let openLinkInNewTab: boolean = $state(false);

    $effect(() => {
        value;
        untrack(() => {
            if (value !== editor?.getHTML()) {
                editor?.commands.setContent(value);
            }
        });
    });

    onMount(() => {
        editor = new Editor({
            element: element,
            extensions: [
                StarterKit,
                Underline,
                Placeholder.configure({
                    placeholder: placeholder,
                }),
                Mention.configure({
                    HTMLAttributes: {
                        class: "mention",
                    },
                    renderHTML({ node, options }) {
                        return [
                            "a",
                            mergeAttributes(
                                { href: `/profile/@${options.HTMLAttributes["data-label"]}` },
                                options.HTMLAttributes,
                            ),
                            options.renderText?.({
                                options: options,
                                node,
                            }),
                        ];
                    },
                    suggestion: {
                        allowToIncludeChar: true,
                        items: async ({ query }) => {
                            try {
                                const actors: Actor[] = await searchActors(
                                    query,
                                    true,
                                );
                                return actors.map((a) => ({
                                    text: a.username!,
                                    description: `@${a.preferred_username}${a.isLocal ? "" : "@" + a.domain}`,
                                    value: a,
                                    icon:
                                        a.icon ||
                                        `https://api.dicebear.com/7.x/initials/svg?seed=${a.preferred_username}&backgroundType=gradientLinear`,
                                }));
                            } catch (e) {
                                console.error(e);
                                show_toast({
                                    type: "error",
                                    icon: "close",
                                    text: "Error during search",
                                });
                                return [];
                            }
                        },

                        render: () => {
                            let component: DropdownList;
                            const componentState: any = $state({
                                items: [],
                                id: "mention-list",
                            });

                            function updatePosition({
                                clientRect,
                            }: SuggestionProps) {
                                const searchListElement =
                                    document.getElementById("mention-list");
                                if (!clientRect || !searchListElement) return;
                                const box = clientRect();
                                if (!box) {
                                    return;
                                }
                                
                                searchListElement.style.position = searchListPosition;
                                searchListElement.style.top = `${box.bottom + window.scrollY + 4}px`;
                                searchListElement.style.left = `${box.left + window.scrollX}px`;
                                searchListElement.style.zIndex = "1001";
                            }

                            function onActorClick(props: SuggestionProps) {
                                return (_: Event, item: SearchItem) => {
                                    props.command({
                                        id: item.value.iri,
                                        label: `${item.value.preferred_username}${item.value.isLocal ? "" : "@" + item.value.domain}`,
                                    });
                                };
                            }
                            return {
                                onStart: (props) => {
                                    props.clientRect;
                                    componentState.items = props.items;
                                    componentState.onclick =
                                        onActorClick(props);

                                    component = mount(SearchList, {
                                        target: element,
                                        props: componentState,
                                    });

                                    updatePosition(props);
                                },
                                onUpdate(props) {
                                    componentState.items = props.items;
                                    componentState.onclick =
                                        onActorClick(props);
                                    updatePosition(props);
                                },

                                onKeyDown(props) {
                                    if (props.event.key === "Escape") {
                                        this.onExit?.({} as unknown as any)

                                        return true;
                                    }

                                    return false
                                },
                                onExit() {
                                    unmount(component);
                                },
                            };
                        },
                    },
                }),
                Link.configure({
                    openOnClick: false,
                    autolink: true,
                    defaultProtocol: "https",
                    protocols: ["http", "https"],
                    isAllowedUri: (url, ctx) => {
                        try {
                            const parsedUrl = url.includes(":")
                                ? new URL(url)
                                : new URL(`${ctx.defaultProtocol}://${url}`);

                            if (!ctx.defaultValidate(parsedUrl.href)) {
                                return false;
                            }

                            const disallowedProtocols = [
                                "ftp",
                                "file",
                                "mailto",
                            ];
                            const protocol = parsedUrl.protocol.replace(
                                ":",
                                "",
                            );

                            if (disallowedProtocols.includes(protocol)) {
                                return false;
                            }

                            const allowedProtocols = ctx.protocols.map((p) =>
                                typeof p === "string" ? p : p.scheme,
                            );

                            if (!allowedProtocols.includes(protocol)) {
                                return false;
                            }

                            return true;
                        } catch {
                            return false;
                        }
                    },
                }),
            ],
            content: value,
            onTransaction: ({ editor: newEditor }) => {
                // force re-render so `editor.isActive` works as expected
                editor = undefined;
                editor = newEditor;
            },
            onUpdate: (props) => {
                value = editor?.getHTML() ?? "";
            },
            editorProps: {
                attributes: {
                    class: `prose dark:prose-invert text-content bg-input-background border border-input-border rounded-md p-3 resize-none transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0 ${extraClasses}`,
                },
            },
            onSelectionUpdate: ({ editor }) => {
                if (editor.isActive("paragraph")) {
                    currentFontSize = "p";
                    return;
                }
                for (let i = 1; i <= 5; i++) {
                    if (editor.isActive("heading", { level: i })) {
                        currentFontSize = "h" + i;
                        return;
                    }
                }
            },
        });
    });

    onDestroy(() => {
        if (editor) {
            editor.destroy();
        }
    });

    function openLinkModal() {
        linkURL = editor?.getAttributes("link").href ?? "";
        openLinkInNewTab = editor?.getAttributes("link").target === "_blank";
        if (linkURL) {
            editor?.chain().focus().extendMarkRange("link").run();
        }
        linkText =
            editor?.state.doc.textBetween(
                editor.state.selection.from,
                editor.state.selection.to,
                "",
            ) ?? "";

        modal.openModal();
    }

    function setLink() {
        linkURLError = "";

        if (linkURL === "") {
            linkURLError = $_("required");
            return;
        }
        try {
            z.string().url().parse(linkURL);
            editor
                ?.chain()
                .focus()
                .insertContent({
                    type: "text",
                    text: linkText || linkURL,
                    marks: [
                        {
                            type: "link",
                            attrs: {
                                href: linkURL,
                                target: openLinkInNewTab ? "_blank" : null,
                            },
                        },
                    ],
                })
                .run();

                modal.closeModal();
        } catch (e) {
            if (
                e instanceof ZodError &&
                e.errors[0].code === "invalid_string"
            ) {
                linkURLError = $_("not-a-valid-url");
            }
            console.error(e);
        }
    }

    function unsetLink() {
        editor?.chain().focus().extendMarkRange("link").unsetLink().run();
        modal.closeModal();
    }
</script>

<div id="editor-wrapper">
    {#if label.length}
        <p class="text-sm font-medium mb-1">
            {label}
        </p>
    {/if}
    <!-- Toolbar -->
    <div class="flex flex-wrap items-center py-2 gap-y-2">
        <div class="mr-2">
            <!-- <Select
                items={fontSizes}
                bind:value={currentFontSize}
                onchange={(value: string) => {
                    if (value.startsWith("p")) {
                        editor?.chain().focus().setParagraph().run();
                    } else {
                        editor
                            ?.chain()
                            .focus()
                            .setHeading({
                                level: parseInt(value.substring(1)) as Level,
                            })
                            .run();
                    }
                }}
            ></Select> -->
        </div>
        <div class="flex gap-2 border-r border-input-border pr-2">
            <button
                type="button"
                class="btn-icon"
                onclick={() => editor?.chain().focus().toggleBold().run()}
                class:ring-2={editor?.isActive("bold")}
                aria-label="Bold"
            >
                <i class="fas fa-bold"></i>
            </button>

            <button
                type="button"
                class="btn-icon"
                onclick={() => editor?.chain().focus().toggleItalic().run()}
                class:ring-2={editor?.isActive("italic")}
                aria-label="Italic"
            >
                <i class="fas fa-italic"></i>
            </button>

            <button
                type="button"
                class="btn-icon"
                onclick={() => editor?.chain().focus().toggleUnderline().run()}
                class:ring-2={editor?.isActive("underline")}
                aria-label="Underline"
            >
                <i class="fas fa-underline"></i>
            </button>
        </div>

        <div class="flex gap-1 border-r border-input-border px-2">
            <button
                type="button"
                class="btn-icon"
                onclick={() => editor?.chain().focus().toggleBulletList().run()}
                class:ring-2={editor?.isActive("bulletList")}
                aria-label="Bullet List"
            >
                <i class="fas fa-list-ul"></i>
            </button>

            <button
                type="button"
                class="btn-icon"
                onclick={() =>
                    editor?.chain().focus().toggleOrderedList().run()}
                class:ring-2={editor?.isActive("orderedList")}
                aria-label="Numbered List"
            >
                <i class="fas fa-list-ol"></i>
            </button>
        </div>

        <div class="flex gap-1 border-r border-input-border px-2">
            <button
                type="button"
                class="btn-icon"
                onclick={() => editor?.chain().focus().toggleBlockquote().run()}
                class:ring-2={editor?.isActive("blockquote")}
                aria-label="Quote"
            >
                <i class="fas fa-quote-right"></i>
            </button>
        </div>

        <div class="flex gap-1 px-2">
            <button
                type="button"
                class="btn-icon"
                onclick={() => openLinkModal()}
                class:ring-2={editor?.isActive("link")}
                aria-label="Link"
            >
                <i class="fas fa-link"></i>
            </button>
        </div>
    </div>
    <div bind:this={element}></div>

    {#if error}
        <span class="editor-error text-xs text-red-400">
            {error instanceof Array ? $_(error[0]) : error}
        </span>
    {/if}
</div>
<Modal
    id="editor-modal"
    title={"Insert/edit link"}
    size="md:min-w-lg"
    bind:this={modal}
>
    {#snippet content()}
        <TextField label={"URL"} bind:value={linkURL} error={linkURLError}
        ></TextField>
        <TextField label={$_("text")} bind:value={linkText}></TextField>
        <Toggle
            label={$_("open-in-new-tab", { values: { n: 2 } })}
            bind:value={openLinkInNewTab}
        ></Toggle>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            {#if editor?.getAttributes("link").href}
                <button
                    type="button"
                    class="btn-secondary shrink-0"
                    onclick={() => unsetLink()}
                    aria-label="Link"
                >
                    <i class="fas fa-link-slash"></i>
                    <span>{$_("unlink")}</span>
                </button>
                <div class="basis-full"></div>
            {/if}
            <button
                type="button"
                class="btn-secondary"
                onclick={() => modal.closeModal()}>{$_("cancel")}</button
            >
            <button
                class="btn-primary"
                type="button"
                name="save"
                onclick={() => setLink()}>{$_("save")}</button
            >
        </div>
    {/snippet}
</Modal>

<style>
    :global(.ProseMirror p.is-editor-empty:first-child::before) {
        content: attr(data-placeholder);
        float: left;
        color: #adb5bd;
        pointer-events: none;
        height: 0;
    }
</style>
