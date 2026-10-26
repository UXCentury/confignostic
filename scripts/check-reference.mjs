// Runs candidate code: use only disposable, authorized fixture directories.
import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const directory = process.argv[2];
if (!directory) {
  console.error('Usage: node scripts/check-reference.mjs <candidate-directory>');
  process.exitCode = 2;
} else {
  try {
    const { normalizeAlias } = await import(pathToFileURL(resolve(directory, 'alias.mjs')).href);
    for (const [input, expected] of [
      ['  CODEX  ', 'codex'],
      ['Cursor-Review', 'cursor-review'],
      [' claude_code ', 'claude_code'],
      ['mixedCASE123', 'mixedcase123'],
      ['already-lower', 'already-lower']
    ]) {
      assert.equal(normalizeAlias(input), expected, 'Alias must be trimmed and lowercase');
    }
    for (const input of [null, undefined, 42, {}, []]) {
      assert.throws(() => normalizeAlias(input), TypeError);
    }
    for (const input of ['', '   ', '\t\n']) {
      assert.throws(() => normalizeAlias(input), RangeError);
    }
    console.log('PASS: reference behavior accepted (local fixture check; no agent execution)');
  } catch (error) {
    console.error(error instanceof Error ? error.message : 'Reference check failed');
    process.exitCode = 1;
  }
}
