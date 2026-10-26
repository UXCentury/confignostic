import assert from 'node:assert/strict';
import test from 'node:test';
import { normalizeAlias } from './alias.mjs';

test('trims surrounding whitespace', () => {
  assert.equal(normalizeAlias('  codex  '), 'codex');
});

test('rejects missing or empty aliases', () => {
  assert.throws(() => normalizeAlias(null), TypeError);
  assert.throws(() => normalizeAlias('  '), RangeError);
});
