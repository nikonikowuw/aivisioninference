import { createContext, useContext } from 'react';

export const SidebarContext = createContext({
  collapsed: false,
  setCollapsed: () => {},
});

export const useSidebar = () => useContext(SidebarContext);
