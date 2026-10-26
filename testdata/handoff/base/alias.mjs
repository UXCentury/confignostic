// Deliberately incomplete reference task: aliases must also be lowercase.
export function normalizeAlias(input) {
  if (typeof input !== 'string') throw new TypeError('Alias must be a string');
  const value = input.trim();
  if (!value) throw new RangeError('Alias must not be empty');
  return value;
}
