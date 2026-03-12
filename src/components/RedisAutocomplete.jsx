/* @refresh reset */
import { useState, useEffect, useRef, useCallback } from 'react';
import { createPortal } from 'react-dom';
import Editor from 'react-simple-code-editor';
import Prism from 'prismjs';
import 'prismjs/components/prism-sql'; // We'll use SQL for basic highlighting or customize it
import { getTextareaCaretViewportPosition } from '../utils/textareaCaret'
import { useBoxSelection } from '../hooks/useBoxSelection';

const REDIS_COMMANDS = [
  { label: 'GET', insert: 'GET ', hint: 'key', description: 'Get the value of a key' },
  { label: 'SET', insert: 'SET ', hint: 'key value', description: 'Set the string value of a key' },
  { label: 'DEL', insert: 'DEL ', hint: 'key', description: 'Delete a key' },
  { label: 'EXISTS', insert: 'EXISTS ', hint: 'key', description: 'Determine if a key exists' },
  { label: 'EXPIRE', insert: 'EXPIRE ', hint: 'key seconds', description: "Set a key's time to live in seconds" },
  { label: 'KEYS', insert: 'KEYS ', hint: 'pattern', description: 'Find all keys matching a pattern' },
  { label: 'SCAN', insert: 'SCAN ', hint: 'cursor [MATCH pattern] [COUNT count]', description: 'Incrementally iterate the keys space' },
  { label: 'FLUSHALL', insert: 'FLUSHALL', hint: '', description: 'Remove all keys from all databases' },
  { label: 'HGET', insert: 'HGET ', hint: 'key field', description: 'Get the value of a hash field' },
  { label: 'HSET', insert: 'HSET ', hint: 'key field value', description: 'Set the string value of a hash field' },
  { label: 'HGETALL', insert: 'HGETALL ', hint: 'key', description: 'Get all the fields and values in a hash' },
  { label: 'LPUSH', insert: 'LPUSH ', hint: 'key value', description: 'Prepend a value to a list' },
  { label: 'RPUSH', insert: 'RPUSH ', hint: 'key value', description: 'Append a value to a list' },
  { label: 'LPOP', insert: 'LPOP ', hint: 'key', description: 'Remove and get the first element in a list' },
  { label: 'RPOP', insert: 'RPOP ', hint: 'key', description: 'Remove and get the last element in a list' },
  { label: 'SADD', insert: 'SADD ', hint: 'key member', description: 'Add a member to a set' },
  { label: 'SMEMBERS', insert: 'SMEMBERS ', hint: 'key', description: 'Get all the members in a set' },
  { label: 'ZADD', insert: 'ZADD ', hint: 'key score member', description: 'Add a member to a sorted set' },
  { label: 'ZRANGE', insert: 'ZRANGE ', hint: 'key start stop', description: 'Return a range of members in a sorted set' },
  { label: 'PUBLISH', insert: 'PUBLISH ', hint: 'channel message', description: 'Post a message to a channel' },
  { label: 'SUBSCRIBE', insert: 'SUBSCRIBE ', hint: 'channel', description: 'Listen for messages published to a channel' },
  { label: 'INFO', insert: 'INFO', hint: '[section]', description: 'Get information and statistics about the server' },
  { label: 'CONFIG GET', insert: 'CONFIG GET ', hint: 'parameter', description: 'Get the value of a configuration parameter' },
];

export function RedisAutocomplete({ value, onChange, onKeyDown, onContextMenu, disabled, placeholder, style }) {
  const [suggestions, setSuggestions] = useState([]);
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [open, setOpen] = useState(false);
  const [dropdownPos, setDropdownPos] = useState(null);
  const [editorRoot, setEditorRoot] = useState(null);
  const containerRef = useRef(null);
  const valueRef = useRef(value);
  useEffect(() => { valueRef.current = value; }, [value]);

  const boxSel = useBoxSelection({
    containerRef,
    valueRef,
    onChange,
    disabled,
    onActivate: () => {
      setOpen(false);
      setDropdownPos(null);
    },
  });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const find = () => container.querySelector('.query-editor-container');
    const el = find();
    if (el) { setEditorRoot(el); return; }
    const obs = new MutationObserver(() => {
      const next = find();
      if (next) { setEditorRoot(next); obs.disconnect(); }
    });
    obs.observe(container, { childList: true, subtree: true });
    return () => obs.disconnect();
  }, []);

  const buildSuggestions = (text) => {
    const parts = text.split(/\s+/);
    const lastPart = parts[parts.length - 1].toUpperCase();
    
    if (!lastPart) return [];
    
    return REDIS_COMMANDS.filter(cmd => 
      cmd.label.startsWith(lastPart)
    ).slice(0, 10);
  };

  const handleKeyUp = (e) => {
    if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', 'Tab', 'Escape'].includes(e.key)) return;
    
    const list = buildSuggestions(value);
    if (list.length > 0) {
      setSuggestions(list);
      setSelectedIdx(0);
      setOpen(true);
      const ta = e.target;
      const caret = getTextareaCaretViewportPosition(ta, ta.selectionStart);
      if (caret) {
        const desiredTop = caret.top + caret.height + 6;
        const desiredLeft = caret.left;
        const maxLeft = Math.max(8, window.innerWidth - 420);
        const maxTop = Math.max(8, window.innerHeight - 300);
        setDropdownPos({
          top: Math.min(desiredTop, maxTop),
          left: Math.min(desiredLeft, maxLeft),
        });
      }
    } else {
      setOpen(false);
      setDropdownPos(null);
    }
  };

  const insertSuggestion = (s) => {
    const parts = value.split(/\s+/);
    parts[parts.length - 1] = s.insert;
    onChange(parts.join(' '));
    setOpen(false);
    setDropdownPos(null);
  };

  const onLocalKeyDown = (e) => {
    if (open) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => (i + 1) % suggestions.length); return; }
      if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => (i - 1 + suggestions.length) % suggestions.length); return; }
      if (e.key === 'Tab' || e.key === 'Enter') {
        e.preventDefault();
        insertSuggestion(suggestions[selectedIdx]);
        return;
      }
      if (e.key === 'Escape') { setOpen(false); return; }
    }
    onKeyDown?.(e);
  };

  // Close on click outside
  useEffect(() => {
    const close = (e) => {
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setOpen(false);
        setDropdownPos(null);
      }
    };
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, []);

  // Attach context menu handler to underlying textarea (react-simple-code-editor wraps a textarea)
  useEffect(() => {
    if (!onContextMenu) return;
    const container = containerRef.current;
    if (!container) return;
    const attach = () => {
      const ta = container.querySelector('textarea');
      if (!ta) return null;
      ta.dataset.darubeEditorRole = 'redis';
      ta.addEventListener('contextmenu', onContextMenu);
      return ta;
    };
    let ta = attach();
    if (!ta) {
      const obs = new MutationObserver(() => {
        ta = attach();
        if (ta) obs.disconnect();
      });
      obs.observe(container, { childList: true, subtree: true });
      return () => obs.disconnect();
    }
    return () => { if (ta) ta.removeEventListener('contextmenu', onContextMenu); };
  }, [onContextMenu]);

  return (
    <div ref={containerRef} className="redis-autocomplete-container">
      <Editor
        value={value}
        onValueChange={onChange}
        highlight={code => Prism.highlight(code, Prism.languages.sql, 'sql')}
        padding={16}
        className="query-editor-container"
        textareaClassName="query-editor-textarea"
        onKeyDown={onLocalKeyDown}
        onKeyUp={handleKeyUp}
        disabled={disabled}
        placeholder={placeholder || 'Enter Redis command (e.g. GET mykey)'}
        style={style}
      />

      {editorRoot && boxSel.active && boxSel.overlay && createPortal(
        <div className="box-selection-overlay" aria-hidden="true">
          {boxSel.overlay.map((r) => (
            <div
              key={`box-${r.row}`}
              className="box-selection-rect"
              style={{ top: `${r.top}px`, left: `${r.left}px`, width: `${r.width}px`, height: `${r.height}px` }}
            />
          ))}
        </div>,
        editorRoot
      )}

      {open && dropdownPos && createPortal(
        <ul
          className="ac-dropdown redis-ac-dropdown"
          style={{ position: 'fixed', top: `${dropdownPos.top}px`, left: `${dropdownPos.left}px` }}
          onMouseDown={e => e.preventDefault()}
        >
          {suggestions.map((s, i) => (
            <li
              key={s.label}
              className={`ac-item ${i === selectedIdx ? 'ac-selected' : ''}`}
              style={{ display: 'block', padding: '8px 12px' }}
              onMouseEnter={() => setSelectedIdx(i)}
              onClick={() => insertSuggestion(s)}
            >
              <div className="ac-main">
                <span className="ac-label">{s.label}</span>
                <span className="ac-hint">{s.hint}</span>
              </div>
              <div className="ac-desc">{s.description}</div>
            </li>
          ))}
        </ul>,
        document.body
      )}
    </div>
  );
}
