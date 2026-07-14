## 0.0.1

- Initial alpha release.
- Headless Tiptap engine running in an invisible WebView.
- Native Flutter document rendering with position-accurate selection and cursor painting.
- Composable widget API: `EditorController`, `TiptapEditor`, `TiptapToolbar`, `DebugOverlay`.
- Delta-based text input with hardware keyboard fallback.
- Extensible node renderer registry for custom node types.
- Full content round-trip: load HTML/JSON, edit, retrieve as HTML, JSON, or plain text.

## 0.0.2

- Fix pub.dev analysis issues.

## 0.0.3

- Fix: Correct cursor placement in empty paragraphs.
- Fix: Remove pan-to-select text.
- Feat: Images support.

## 0.0.4

- Update README.md

## 0.0.5

- Refactor: Remove rendering for dropped engine extensions.
- Feat: Replace debug overlay, with performance overlay.
  - **Breaking:** `DebugOverlay` is removed. Use `TiptapPerformanceOverlay` instead.
- Refactor: Replace hardcoded strings with named constants.
- Refactor: Add typed query-result classes and route controller queries through them.
- Refactor: Split protocol_types into per-concern files.
- Refactor: Split document_renderer builders into part files.
- Refactor: Extract block-text extractor and typing-latency tracker.

## 0.0.6

- Update README.md

## 0.1.0

- Feat: Select text, copy to clipboard.

## 0.1.1

- Update README.md
- Better performance metrices.
- Better engine performance.
