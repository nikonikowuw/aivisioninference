// Chakra imports
import {
  Box,
  Portal,
  Text,
  useColorModeValue,
  useDisclosure,
} from "@chakra-ui/react";
// Layout components
import Navbar from "components/navbar/NavbarAdmin";
import Sidebar from "components/sidebar/Sidebar";
import { useAuth } from "contexts/AuthContext";
import { SidebarContext } from "contexts/SidebarContext";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { hasAnyPermission } from "utils/permission";
import {
  generateRoutesFromMenus,
  generateSidebarRoutesFromMenus,
  getActiveRouteFromMenus,
} from "../../router";
import PersonDetail from "../../views/admin/persons/detail";
import EdgeNodeDetail from "../../views/admin/devices/edge-nodes/detail";

export default function Dashboard(props: { [x: string]: any }) {
  const { ...rest } = props;
  const fixed = false;
  const [collapsed, setCollapsed] = useState(false);
  const { user } = useAuth();
  const { t } = useTranslation(["layout", "menu"]);
  const location = useLocation();
  const noAccessColor = useColorModeValue("gray.500", "gray.400");

  const menus = useMemo(() => {
    const rawMenus = user?.menus || [];
    const permissionCodes = user?.permission_codes || [];
    const canViewSmartRecords = hasAnyPermission(permissionCodes, [
      "records:recognition:list",
      "records:alarm:list",
      "records:capture:list",
    ]);
    const hasSmartRecordsMenu = rawMenus.some(
      (menu) => menu.code === "smart-records",
    );
    if (!canViewSmartRecords) {
      return rawMenus.filter((menu) => menu.code !== "smart-records");
    }
    if (hasSmartRecordsMenu) return rawMenus;
    return [
      ...rawMenus,
      {
        id: "smart-records",
        name: t("menu:smart-records"),
        code: "smart-records",
        path: "/smart-records",
        icon: "MdNotificationsActive",
        sort_order: 35,
      },
    ];
  }, [user?.menus, user?.permission_codes, t]);

  // 从用户菜单生成侧边栏路由
  const sidebarRoutes = useMemo(() => {
    if (menus.length > 0) {
      return generateSidebarRoutesFromMenus(menus, t);
    }
    return [];
  }, [menus, t]);

  // 从用户菜单生成路由组件
  const dynamicRoutes = useMemo(() => {
    if (menus.length > 0) {
      return generateRoutesFromMenus(menus);
    }
    return [];
  }, [menus]);

  // 获取当前激活的路由名称
  const brandText = useMemo(() => {
    if (menus.length > 0) {
      return getActiveRouteFromMenus(menus, location.pathname, t);
    }
    return t("menu:dashboard");
  }, [menus, location.pathname, t]);

  useEffect(() => {
    document.documentElement.dir = "ltr";
  }, []);

  const getRoute = () => {
    return location.pathname !== "/admin/full-screen-maps";
  };

  const { onOpen } = useDisclosure();

  const sidebarWidth = collapsed ? "80px" : "260px";

  return (
    <Box>
      <SidebarContext.Provider
        value={{
          collapsed,
          setCollapsed,
          sidebarRoutes,
        }}
      >
        <Sidebar routes={sidebarRoutes} display="none" {...rest} />
        <Box
          float="right"
          minHeight="100vh"
          height="100%"
          overflow="auto"
          position="relative"
          maxHeight="100%"
          w={{ base: "100%", xl: `calc(100% - ${sidebarWidth})` }}
          maxWidth={{ base: "100%", xl: `calc(100% - ${sidebarWidth})` }}
          transition="all 0.33s cubic-bezier(0.685, 0.0473, 0.346, 1)"
          transitionDuration=".2s, .2s, .35s"
          transitionProperty="top, bottom, width"
          transitionTimingFunction="linear, linear, ease"
        >
          <Portal>
            <Box>
              <Navbar
                onOpen={onOpen}
                logoText={t("navbar.logoText")}
                brandText={brandText}
                secondary={false}
                message={brandText}
                fixed={fixed}
                {...rest}
              />
            </Box>
          </Portal>

          {getRoute() && (
            <Box
              mx="auto"
              p={{ base: "20px", md: "30px" }}
              pe="20px"
              minH="100vh"
              pt="50px"
            >
              {dynamicRoutes.length === 0 ? (
                <Text color={noAccessColor} textAlign="center" mt="40px">
                  {t("noAccess")}
                </Text>
              ) : (
                <Routes>
                  {/* 静态关键路由置顶，且不再依赖细颗粒度权限判断（由组件内部处理），保证跳转稳定性 */}
                  <Route path="persons/:id" element={<PersonDetail />} />
                  <Route path="devices/edge-nodes/:id" element={<EdgeNodeDetail />} />

                  {dynamicRoutes}
                  <Route
                    path="/"
                    element={<Navigate to="/admin/default" replace />}
                  />
                </Routes>
              )}
            </Box>
          )}
        </Box>
      </SidebarContext.Provider>
    </Box>
  );
}
