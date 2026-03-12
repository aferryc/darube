import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import CssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
import HtmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import TsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';

// Language + feature contributions (tree-shaken by bundler).
import 'monaco-editor/esm/vs/basic-languages/sql/sql.contribution';
import 'monaco-editor/esm/vs/language/typescript/monaco.contribution';

export function applyDarubeTheme(monaco) {
  // Idempotent theme registration.
  if (!monaco?.editor?.defineTheme) return;
  if (globalThis.__darubeMonacoThemeApplied) return;
  globalThis.__darubeMonacoThemeApplied = true;

  // Keep this aligned with `src/styles/Variables.css`.
  const BG_EDITOR = '#0d1424';
  const BG_CARD = '#162033';
  const BG_DARK = '#0b1220';
  const TEXT = '#e6edf6';
  const TEXT_MUTED = '#94a3b8';
  const BORDER = '#1e2b45';
  const ACCENT = '#4f8cff';

  // Legacy Prism palette used previously in Darube editors.
  const TOK_KEYWORD = '#c678dd';
  const TOK_STRING = '#98c379';
  const TOK_NUMBER = '#d19a66';
  const TOK_FUNC = '#61afef';
  const TOK_OP = '#56b6c2';
  const TOK_PUNC = '#abb2bf';
  const TOK_COMMENT = '#5c6370';

  monaco.editor.defineTheme('darube-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: TOK_KEYWORD.slice(1) },
      { token: 'keyword.sql', foreground: TOK_KEYWORD.slice(1) },
      { token: 'string', foreground: TOK_STRING.slice(1) },
      { token: 'string.sql', foreground: TOK_STRING.slice(1) },
      { token: 'number', foreground: TOK_NUMBER.slice(1) },
      { token: 'number.sql', foreground: TOK_NUMBER.slice(1) },
      { token: 'function', foreground: TOK_FUNC.slice(1) },
      { token: 'operator', foreground: TOK_OP.slice(1) },
      { token: 'delimiter', foreground: TOK_PUNC.slice(1) },
      { token: 'delimiter.parenthesis', foreground: TOK_PUNC.slice(1) },
      { token: 'delimiter.sql', foreground: TOK_PUNC.slice(1) },
      { token: 'comment', foreground: TOK_COMMENT.slice(1), fontStyle: 'italic' },
      { token: 'comment.sql', foreground: TOK_COMMENT.slice(1), fontStyle: 'italic' },
    ],
    colors: {
      // Core editor
      'editor.background': BG_EDITOR,
      'editor.foreground': TEXT,
      'editorCursor.foreground': ACCENT,
      'editorLineNumber.foreground': TEXT_MUTED,
      'editorLineNumber.activeForeground': TEXT,
      'editorIndentGuide.background1': BORDER,
      'editorIndentGuide.activeBackground1': TEXT_MUTED,
      'editorWhitespace.foreground': BORDER,
      'editor.selectionBackground': `${ACCENT}33`,
      'editor.inactiveSelectionBackground': `${ACCENT}22`,
      'editor.lineHighlightBackground': '#0f1a33',
      'editor.findMatchBackground': `${ACCENT}33`,
      'editor.findMatchHighlightBackground': `${ACCENT}22`,

      // Widgets
      'editorWidget.background': BG_CARD,
      'editorWidget.border': BORDER,
      'editorSuggestWidget.background': BG_CARD,
      'editorSuggestWidget.border': BORDER,
      'editorSuggestWidget.foreground': TEXT,
      'editorSuggestWidget.highlightForeground': ACCENT,
      'editorSuggestWidget.selectedBackground': `${ACCENT}2b`,
      'editorHoverWidget.background': BG_CARD,
      'editorHoverWidget.border': BORDER,

      // Scrollbars
      'scrollbarSlider.background': '#2a3a5a55',
      'scrollbarSlider.hoverBackground': '#2a3a5a88',
      'scrollbarSlider.activeBackground': '#2a3a5abb',

      // Diff / peek / misc
      'peekView.border': BORDER,
      'peekViewEditor.background': BG_DARK,
      'peekViewResult.background': BG_DARK,
      'peekViewTitle.background': BG_CARD,
      'list.activeSelectionBackground': `${ACCENT}2b`,
      'list.inactiveSelectionBackground': `${ACCENT}1f`,
      'list.hoverBackground': `${ACCENT}14`,
      'list.highlightForeground': ACCENT,
      'focusBorder': BORDER,
    },
  });
}

// Configure web workers for Monaco (Vite `?worker` imports above).
// Must run before creating any editor instances.
if (typeof globalThis !== 'undefined' && !globalThis.MonacoEnvironment) {
  globalThis.MonacoEnvironment = {
    getWorker(_workerId, label) {
      if (label === 'json') return new JsonWorker();
      if (label === 'css' || label === 'scss' || label === 'less') return new CssWorker();
      if (label === 'html' || label === 'handlebars' || label === 'razor') return new HtmlWorker();
      if (label === 'typescript' || label === 'javascript') return new TsWorker();
      return new EditorWorker();
    },
  };
}
