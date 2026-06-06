import { describe, it, expect } from 'vitest';
import { hasPermission, hasAnyPermission } from './permission';

describe('hasPermission', () => {
  it('returns false for null/undefined/empty codes', () => {
    expect(hasPermission(null, 'user:list')).toBe(false);
    expect(hasPermission(undefined, 'user:list')).toBe(false);
    expect(hasPermission([], 'user:list')).toBe(false);
  });

  it('returns true when the code is in the list', () => {
    expect(hasPermission(['user:list', 'user:create'], 'user:list')).toBe(true);
  });

  it('returns false when the code is not in the list', () => {
    expect(hasPermission(['user:list'], 'user:create')).toBe(false);
  });

  it('returns true for any code when wildcard * is present', () => {
    expect(hasPermission(['*'], 'anything:goes')).toBe(true);
  });
});

describe('hasAnyPermission', () => {
  it('returns false for null/undefined/empty codes', () => {
    expect(hasAnyPermission(null, ['a', 'b'])).toBe(false);
    expect(hasAnyPermission(undefined, ['a', 'b'])).toBe(false);
    expect(hasAnyPermission([], ['a', 'b'])).toBe(false);
  });

  it('returns true if any required code is present', () => {
    expect(hasAnyPermission(['records:alarm:list'], ['records:recognition:list', 'records:alarm:list'])).toBe(true);
  });

  it('returns false if none of the required codes is present', () => {
    expect(hasAnyPermission(['user:list'], ['records:recognition:list', 'records:alarm:list'])).toBe(false);
  });

  it('returns true for wildcard *', () => {
    expect(hasAnyPermission(['*'], ['records:recognition:list'])).toBe(true);
  });
});
