import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { cpSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const fixture = dirname(fileURLToPath(import.meta.url));
const checker = resolve(fixture, '../../scripts/check-reference.mjs');

function run(args) {
  const result = spawnSync(process.execPath, args, { encoding: 'utf8', timeout: 10000 });
  assert.ifError(result.error);
  return result;
}

test('baseline passes its original tests but fails the new acceptance requirement', () => {
  const original = run(['--test', join(fixture, 'base/alias.test.mjs')]);
  assert.equal(original.status, 0, original.stdout + original.stderr);
  const acceptance = run([checker, join(fixture, 'base')]);
  assert.equal(acceptance.status, 1, acceptance.stdout + acceptance.stderr);
  assert.match(acceptance.stderr, /trimmed and lowercase/);
});

test('the bounded reference change passes both original and acceptance tests', t => {
  const candidate = mkdtempSync(join(tmpdir(), 'confignostic-reference-'));
  t.after(() => rmSync(candidate, { recursive: true, force: true }));
  cpSync(join(fixture, 'base'), candidate, { recursive: true });
  const path = join(candidate, 'alias.mjs');
  const source = readFileSync(path, 'utf8');
  assert.equal(source.split('return value;').length, 2);
  writeFileSync(path, source.replace('return value;', 'return value.toLowerCase();'));
  const original = run(['--test', join(candidate, 'alias.test.mjs')]);
  assert.equal(original.status, 0, original.stdout + original.stderr);
  const acceptance = run([checker, candidate]);
  assert.equal(acceptance.status, 0, acceptance.stdout + acceptance.stderr);
});
