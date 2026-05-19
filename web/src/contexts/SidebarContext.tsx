import { createContext, useContext, Dispatch, SetStateAction } from 'react';
import type { SidebarRouteType } from '../router/types';

export interface SidebarContextType {
  collapsed: boolean;
  setCollapsed: Dispatch<SetStateAction<boolean>>;
  sidebarRoutes: SidebarRouteType[];
}

export const SidebarContext = createContext<SidebarContextType>({
  collapsed: false,
  setCollapsed: () => {},
  sidebarRoutes: [],
});

export const useSidebar = () => useContext(SidebarContext);
