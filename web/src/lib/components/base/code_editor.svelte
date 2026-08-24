<script lang="ts">
    import { syntaxHighlighting } from "@codemirror/language";
    import { Compartment, EditorState } from "@codemirror/state";
    import { EditorView } from "@codemirror/view";
    import { classHighlighter } from "@lezer/highlight";
    import {
        codeEditorLanguage,
        type CodeEditorLanguageDefinition,
    } from "$lib/util/code_editor_language";
    import { basicSetup } from "codemirror";
    import { onDestroy, onMount } from "svelte";

    interface Props {
        value?: string;
        label?: string;
        ariaLabel?: string;
        error?: string | string[] | null;
        language?: CodeEditorLanguageDefinition;
    }

    let {
        value = $bindable(""),
        label = "",
        ariaLabel = "",
        error = "",
        language,
    }: Props = $props();

    let element: HTMLDivElement;
    let editor: EditorView | undefined;
    const languageCompartment = new Compartment();

    const editorTheme = EditorView.theme({
        "&": {
            backgroundColor: "rgba(var(--input-background))",
            color: "rgba(var(--content))",
            border: "1px solid rgba(var(--input-border))",
            borderRadius: "0.375rem",
            fontSize: "0.75rem",
        },
        "&.cm-focused": {
            outline: "none",
            borderColor: "rgba(var(--input-border-focus))",
        },
        ".cm-scroller": {
            fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
            lineHeight: "1.5rem",
            maxHeight: "min(58vh, 44rem)",
            overflow: "auto",
        },
        ".cm-content": {
            caretColor: "rgba(var(--content))",
            minHeight: "min(40vh, 28rem)",
            padding: "0.75rem 0",
        },
        ".cm-line": {
            padding: "0 0.75rem",
        },
        ".cm-gutters": {
            backgroundColor: "rgba(var(--menu-background))",
            color: "rgb(107 114 128)",
            borderRight: "1px solid rgba(var(--input-border))",
        },
        ".cm-gutter": {
            minHeight: "min(40vh, 28rem)",
        },
        ".cm-activeLine, .cm-activeLineGutter": {
            backgroundColor: "rgba(var(--menu-item-background-hover), 0.55)",
        },
        "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
            backgroundColor: "rgba(var(--input-ring), 0.35)",
        },
        ".cm-cursor, .cm-dropCursor": {
            borderLeftColor: "rgba(var(--content))",
        },
        ".cm-foldPlaceholder": {
            backgroundColor: "rgba(var(--menu-item-background-focus))",
            borderColor: "rgba(var(--input-border))",
            color: "rgba(var(--content))",
        },
        ".cm-tooltip": {
            backgroundColor: "rgba(var(--menu-background))",
            borderColor: "rgba(var(--input-border))",
            color: "rgba(var(--content))",
        },
        ".cm-panels": {
            backgroundColor: "rgba(var(--menu-background))",
            color: "rgba(var(--content))",
        },
    });

    onMount(() => {
        editor = new EditorView({
            parent: element,
            state: EditorState.create({
                doc: value,
                extensions: [
                    basicSetup,
                    syntaxHighlighting(classHighlighter),
                    languageCompartment.of(codeEditorLanguage(language)),
                    EditorView.contentAttributes.of({
                        "aria-label": ariaLabel || label,
                        spellcheck: "false",
                    }),
                    EditorView.updateListener.of((update) => {
                        if (!update.docChanged) return;
                        const nextValue = update.state.doc.toString();
                        if (value !== nextValue) value = nextValue;
                    }),
                    editorTheme,
                ],
            }),
        });
    });

    onDestroy(() => editor?.destroy());

    $effect(() => {
        const nextValue = value;
        if (!editor || editor.state.doc.toString() === nextValue) return;
        editor.dispatch({
            changes: { from: 0, to: editor.state.doc.length, insert: nextValue },
        });
    });

    $effect(() => {
        const nextLanguage = language;
        if (!editor) return;
        editor.dispatch({
            effects: languageCompartment.reconfigure(codeEditorLanguage(nextLanguage)),
        });
    });
</script>

<div class="code-editor">
    {#if label || language?.name || language?.id}
        <div class="mb-1 flex items-center justify-between gap-3">
            {#if label}<p class="text-sm font-medium">{label}</p>{/if}
            {#if language?.name || language?.id}
                <span class="truncate text-xs text-gray-500">
                    {language.name || language.id}
                </span>
            {/if}
        </div>
    {/if}
    <div bind:this={element} class:error={(error?.length ?? 0) > 0}></div>
    {#if error}
        <span class="textfield-error text-xs text-red-400">
            {error instanceof Array ? error[0] : error}
        </span>
    {/if}
</div>

<style>
    .code-editor :global(.tok-keyword) {
        color: #8b5cf6;
        font-weight: 600;
    }

    .code-editor :global(.tok-bool),
    .code-editor :global(.tok-atom) {
        color: #0284c7;
        font-weight: 600;
    }

    .code-editor :global(.tok-number) {
        color: #d97706;
    }

    .code-editor :global(.tok-string) {
        color: #059669;
    }

    .code-editor :global(.tok-typeName) {
        color: #2563eb;
    }

    .code-editor :global(.tok-operator) {
        color: #db2777;
    }

    .code-editor :global(.tok-comment) {
        color: #6b7280;
        font-style: italic;
    }

    .code-editor :global(.tok-meta) {
        color: #dc2626;
        font-weight: 600;
    }

    .code-editor :global(.error .cm-editor) {
        border-color: #f87171;
        background-color: rgba(var(--input-background-error));
    }

    :global(.dark) .code-editor :global(.tok-keyword) {
        color: #c4b5fd;
    }

    :global(.dark) .code-editor :global(.tok-bool),
    :global(.dark) .code-editor :global(.tok-atom) {
        color: #7dd3fc;
    }

    :global(.dark) .code-editor :global(.tok-number) {
        color: #fbbf24;
    }

    :global(.dark) .code-editor :global(.tok-string) {
        color: #6ee7b7;
    }

    :global(.dark) .code-editor :global(.tok-typeName) {
        color: #93c5fd;
    }

    :global(.dark) .code-editor :global(.tok-operator) {
        color: #f9a8d4;
    }

    :global(.dark) .code-editor :global(.tok-comment) {
        color: #9ca3af;
    }

    :global(.dark) .code-editor :global(.tok-meta) {
        color: #fca5a5;
    }
</style>
