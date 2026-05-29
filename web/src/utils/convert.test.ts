import { describe, it, expect } from 'vitest';
import { parseOptionalNumber } from './convert';

describe('parseOptionalNumber', () => {
  it('should return undefined for undefined', () => {
    expect(parseOptionalNumber(undefined)).toBeUndefined();
  });

  it('should return undefined for empty string', () => {
    expect(parseOptionalNumber('')).toBeUndefined();
  });

  it('should return undefined for invalid number string', () => {
    expect(parseOptionalNumber('abc')).toBeUndefined();
    expect(parseOptionalNumber('123a')).toBeUndefined();
  });

  it('should parse valid numbers', () => {
    expect(parseOptionalNumber('0')).toBe(0);
    expect(parseOptionalNumber('123')).toBe(123);
    expect(parseOptionalNumber('-45')).toBe(-45);
    expect(parseOptionalNumber('3.14')).toBe(3.14);
  });
});
