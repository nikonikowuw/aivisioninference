import { beforeEach, describe, expect, it } from 'vitest';
import { clearAccessToken, getAccessToken, getErrorMessage, setAccessToken } from './api';

describe('api error messages', () => {
  it('should fall back to server error for unknown codes', () => {
    expect(getErrorMessage(99999)).toBe('Server error');
  });
});

describe('access token storage', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  it('should persist token in localStorage when remember me is enabled', () => {
    setAccessToken('remembered-token', true);

    expect(localStorage.getItem('access_token')).toBe('remembered-token');
    expect(sessionStorage.getItem('access_token')).toBeNull();
    expect(getAccessToken()).toBe('remembered-token');
  });

  it('should store token in sessionStorage when remember me is disabled', () => {
    setAccessToken('session-token', false);

    expect(localStorage.getItem('access_token')).toBeNull();
    expect(sessionStorage.getItem('access_token')).toBe('session-token');
    expect(getAccessToken()).toBe('session-token');
  });

  it('should clear token from both storage locations', () => {
    localStorage.setItem('access_token', 'local-token');
    sessionStorage.setItem('access_token', 'session-token');

    clearAccessToken();

    expect(localStorage.getItem('access_token')).toBeNull();
    expect(sessionStorage.getItem('access_token')).toBeNull();
  });
});
