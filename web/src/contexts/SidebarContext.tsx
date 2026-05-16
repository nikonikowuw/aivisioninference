import { createContext, useContext, Dispatch, SetStateAction } from 'react';

export interface SidebarContextType {
  collapsed: boolean;
  setCollapsed: Dispatch<SetStateAction<boolean>>;
  sidebarRoutes: any[];
}

export const SidebarContext = createContext<SidebarContextType>({
  collapsed: false,
  setCollapsed: () => {},
  sidebarRoutes: [],
});

export const useSidebar = () => useContext(SidebarContext);
