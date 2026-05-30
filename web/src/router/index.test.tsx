import { describe, expect, it } from 'vitest';
import { isValidElement, type ReactElement } from 'react';
import { generateRoutesFromMenus } from './index';
import type { Menu } from '../services/api';

const createMenu = (code: string, path: string): Menu => ({
  id: code,
  name: code,
  code,
  path,
  icon: '',
  sort_order: 0,
});

describe('generateRoutesFromMenus', () => {
  it('should generate routes for mail config and feedback dynamic menus', () => {
    const routes = generateRoutesFromMenus([
      createMenu('mail-config', 'mail-config'),
      createMenu('feedback', 'feedback'),
    ]);

    const paths = routes
      .filter((route): route is ReactElement<{ path: string }> => isValidElement(route))
      .map((route) => route.props.path);

    expect(paths).toContain('mail-config');
    expect(paths).toContain('feedback');
  });
});
