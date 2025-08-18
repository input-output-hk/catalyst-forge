import { NavLink, useLocation } from "react-router-dom";
import { useAppStore } from "@/store/app-store";
import {
  CircleDot,
  Cpu,
  Layers,
  ListTree,
  Lock,
  Logs,
  Settings,
  SquareChartGantt,
  ChevronDown,
  Users,
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarGroupAction,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  SidebarHeader,
  useSidebar,
} from "@/components/ui/sidebar";
import { useState } from "react";
import { BRAND } from "@/lib/brand";
import { LogoMark } from "@/components/brand/LogoMark";
const coreItems = [
  { title: "Dashboard", url: "/", icon: SquareChartGantt },
  { title: "Services", url: "/services", icon: Cpu },
  { title: "Environments", url: "/environments", icon: Layers },
  { title: "Jobs", url: "/jobs", icon: Logs },
];
const securityItems = [
  { title: "Users", url: "/users", icon: Users },
  { title: "Secrets", url: "/secrets", icon: Lock },
  { title: "Audit Log", url: "/audit", icon: ListTree },
  { title: "RBAC Admin", url: "/admin/rbac", icon: Lock },
];
const platformItems = [
  { title: "Settings", url: "/settings", icon: Settings },
  { title: "Auth Flows", url: "/auth-demo", icon: CircleDot },
];

export function AppSidebar() {
  const { state } = useSidebar();
  const collapsed = state === "collapsed";
  const { state: app, actions } = useAppStore();
  const compact = app.flags.compact;
  const location = useLocation();
  const currentPath = location.pathname;
  const [openSecurity, setOpenSecurity] = useState(false);
  const [openPlatform, setOpenPlatform] = useState(true);

  const baseItemCls = `group relative flex items-center gap-2 rounded-md ${compact ? "px-1.5 py-1.5 text-[13px]" : "px-2.5 py-2 text-sm"} focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 transition-colors`;
  // IMPORTANT: Avoid persistent left rails or vertical borders. Do NOT add before: or border-l here.
  const activeItemCls = `${baseItemCls} text-primary font-semibold bg-primary/10 shadow-[inset_3px_0_0_0_hsl(var(--primary)/0.95)]`;
  const inactiveItemCls = `${baseItemCls} text-muted-foreground hover:bg-primary/5 hover:text-primary`;

  return (
    <Sidebar
      className={`${collapsed ? "w-14" : "w-60"} group-data-[side=left]:border-transparent group-data-[side=right]:border-transparent`}
      collapsible="icon"
    >
      <SidebarContent>
        <SidebarHeader className={collapsed ? "p-1" : "h-14 flex items-center px-2"}>
          <NavLink
            to="/"
            className={`flex items-center rounded-md hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring ${collapsed ? "gap-1 px-1.5 py-1.5" : "gap-2 px-2"}`}
          >
            <span
              className={`rounded-full bg-primary/10 ${collapsed ? "p-0.5" : "p-2"} ${!collapsed ? "relative overflow-hidden logo-motif" : ""}`}
            >
              <LogoMark size={collapsed ? 24 : 40} />
            </span>
            {!collapsed && (
              <span className="text-base font-semibold tracking-tight">{BRAND.shortName}</span>
            )}
          </NavLink>
        </SidebarHeader>
        <SidebarSeparator className="opacity-10" />
        <SidebarGroup>
          {!collapsed && (
            <SidebarGroupLabel className="text-xs tracking-wide text-muted-foreground uppercase">
              Core
            </SidebarGroupLabel>
          )}
          <SidebarGroupContent>
            <SidebarMenu>
              {coreItems.map((item) => {
                const active = currentPath === item.url || currentPath.startsWith(item.url + "/");
                return (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      asChild
                      isActive={active}
                      className={active ? activeItemCls : inactiveItemCls}
                    >
                      <NavLink to={item.url} end>
                        <item.icon className="mr-2 h-[18px] w-[18px]" />
                        {!collapsed && <span>{item.title}</span>}
                      </NavLink>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarSeparator className="opacity-10" />
        <SidebarGroup>
          {!collapsed && (
            <>
              <SidebarGroupLabel className="text-xs tracking-wide text-muted-foreground uppercase">
                Security & Governance
              </SidebarGroupLabel>
              <SidebarGroupAction asChild>
                <button
                  aria-label={openSecurity ? "Collapse section" : "Expand section"}
                  onClick={() => setOpenSecurity((v) => !v)}
                >
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${openSecurity ? "rotate-180" : ""}`}
                  />
                </button>
              </SidebarGroupAction>
            </>
          )}
          {openSecurity && (
            <SidebarGroupContent>
              <SidebarMenu>
                {securityItems
                  .filter((item) => {
                    // Hide admin-only items for non-admins
                    const adminOnly = item.url === "/users" || item.url === "/audit" || item.url === "/admin/rbac";
                    return adminOnly ? app.session.roles?.includes("admin") : true;
                  })
                  .map((item) => {
                    const active =
                      currentPath === item.url || currentPath.startsWith(item.url + "/");
                    return (
                      <SidebarMenuItem key={item.title}>
                        <SidebarMenuButton
                          asChild
                          isActive={active}
                          className={active ? activeItemCls : inactiveItemCls}
                        >
                          <NavLink to={item.url} end>
                            <item.icon className="mr-2 h-[18px] w-[18px]" />
                            {!collapsed && <span>{item.title}</span>}
                          </NavLink>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
              </SidebarMenu>
            </SidebarGroupContent>
          )}
        </SidebarGroup>

        <SidebarGroup>
          {!collapsed && (
            <>
              <SidebarGroupLabel className="text-xs tracking-wide text-muted-foreground uppercase">
                Platform
              </SidebarGroupLabel>
              <SidebarGroupAction asChild>
                <button
                  aria-label={openPlatform ? "Collapse section" : "Expand section"}
                  onClick={() => setOpenPlatform((v) => !v)}
                >
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${openPlatform ? "rotate-180" : ""}`}
                  />
                </button>
              </SidebarGroupAction>
            </>
          )}
          {openPlatform && (
            <SidebarGroupContent>
              <SidebarMenu>
                {platformItems.map((item) => {
                  const active = currentPath === item.url || currentPath.startsWith(item.url + "/");
                  return (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton
                        asChild
                        isActive={active}
                        className={active ? activeItemCls : inactiveItemCls}
                      >
                        <NavLink to={item.url} end>
                          <item.icon className="mr-2 h-[18px] w-[18px]" />
                          {!collapsed && <span>{item.title}</span>}
                        </NavLink>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          )}
        </SidebarGroup>

        <div className="mt-auto" />
        <SidebarGroup>
          {!collapsed && (
            <SidebarGroupLabel className="text-xs tracking-wide text-muted-foreground uppercase">
              Utilities
            </SidebarGroupLabel>
          )}
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  className={`${baseItemCls} text-muted-foreground hover:bg-primary/5 hover:text-foreground/90`}
                  onClick={() => actions.toggleFlag("darkMode")}
                  aria-label="Toggle theme"
                >
                  <svg
                    className="mr-2 h-[18px] w-[18px]"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
                  </svg>
                  {!collapsed && <span>Toggle Theme</span>}
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild>
                  <a
                    href="https://docs.lovable.dev/"
                    target="_blank"
                    rel="noreferrer"
                    className={`${baseItemCls} text-muted-foreground hover:bg-primary/5 hover:text-foreground/90`}
                    aria-label="Open documentation"
                  >
                    <span
                      className="mr-2 inline-block h-[18px] w-[18px] rounded-full border border-current"
                      aria-hidden
                    />
                    {!collapsed && <span>Docs</span>}
                  </a>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}
