/* @refresh reset */
import { useCallback } from 'react';

import { MonacoCodeEditor } from './MonacoCodeEditor';

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

function ensureRedisLanguage(monaco) {
  const exists = (monaco.languages.getLanguages?.() || []).some(l => l.id === 'darube-redis');
  if (!exists) monaco.languages.register({ id: 'darube-redis' });
}

export function RedisAutocomplete({ value, onChange, onKeyDown, onContextMenu, disabled, placeholder, style }) {
  const handleMount = useCallback((monaco, editor) => {
    ensureRedisLanguage(monaco);

    const disposable = monaco.languages.registerCompletionItemProvider('darube-redis', {
      triggerCharacters: [' '],
      provideCompletionItems: (model, position) => {
        const text = model.getValue();
        const cursorPos = model.getOffsetAt(position);
        const before = text.slice(0, cursorPos);

        const parts = before.split(/\s+/);
        const lastPart = (parts[parts.length - 1] || '').toUpperCase();
        if (!lastPart) return { suggestions: [] };

        const list = REDIS_COMMANDS.filter(cmd => cmd.label.startsWith(lastPart)).slice(0, 10);
        if (!list.length) return { suggestions: [] };

        const wMatch = before.match(/([^\s,();]+)$/);
        const rawWord = wMatch ? wMatch[1] : '';
        const startOffset = cursorPos - rawWord.length;
        const startPos = model.getPositionAt(startOffset);
        const range = new monaco.Range(startPos.lineNumber, startPos.column, position.lineNumber, position.column);

        return {
          suggestions: list.map((cmd) => ({
            label: cmd.label,
            kind: monaco.languages.CompletionItemKind.Keyword,
            insertText: cmd.insert,
            range,
            detail: cmd.hint || undefined,
            documentation: cmd.description || undefined,
          })),
        };
      },
    });

    try {
      editor.updateOptions({
        quickSuggestions: { other: true, comments: true, strings: true },
        suggestOnTriggerCharacters: true,
      });
    } catch { /* ignore */ }

    return () => disposable?.dispose?.();
  }, []);

  return (
    <MonacoCodeEditor
      value={value}
      onChange={onChange}
      language="darube-redis"
      disabled={disabled}
      placeholder={placeholder || 'Enter Redis command (e.g. GET mykey)'}
      style={style}
      className="query-editor-container"
      editorRole="redis"
      onKeyDown={onKeyDown}
      onContextMenu={onContextMenu}
      onMount={handleMount}
    />
  );
}

