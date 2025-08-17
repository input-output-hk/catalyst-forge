import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { FeatureFlagsPanel } from "./FeatureFlagsPanel";
import { TopBar } from "./TopBar";

export const AppLayout = ({ children }: { children: React.ReactNode }) => {
  return (
    <SidebarProvider style={{ "--sidebar-width-icon": "3.5rem" } as React.CSSProperties}>
      <div className="min-h-screen flex w-full bg-background text-foreground">
        <aside>
          <AppSidebar />
        </aside>
        <div className="flex-1 flex flex-col">
          <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="h-14 flex items-center gap-2 px-3">
              <SidebarTrigger className="mr-1" />
              <TopBar />
            </div>
          </header>
          <main className="flex-1">{children}</main>
        </div>
      </div>
      <FeatureFlagsPanel />
    </SidebarProvider>
  );
};
