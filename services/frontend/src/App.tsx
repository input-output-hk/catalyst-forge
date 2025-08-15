import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { HelmetProvider } from "react-helmet-async";
import { AppStoreProvider } from "@/store/app-store";
import { AppLayout } from "@/components/layout/AppLayout";
import Dashboard from "@/pages/Dashboard";
import Services from "@/pages/Services";
import Environments from "@/pages/Environments";
import Jobs from "@/pages/Jobs";
import Secrets from "@/pages/Secrets";
import AuditLog from "@/pages/AuditLog";
import SettingsProfile from "@/pages/SettingsProfile";
import Profile from "@/pages/Profile";
import Users from "@/pages/Users";
import { lazy, Suspense } from "react";
import NotFound from "./pages/NotFound";
import { AppShell } from "@/components/layout/AppShell";
import Landing from "@/pages/Landing";
import InviteLanding from "@/pages/InviteLanding";
import { RecoveryKeysGate } from "@/components/auth/RecoveryKeysGate";

const queryClient = new QueryClient();

const AuthFlows = lazy(() => import("@/pages/AuthFlows"));
const App = () => (
  <QueryClientProvider client={queryClient}>
    <HelmetProvider>
      <TooltipProvider>
        <AppStoreProvider>
          <Toaster />
          <Sonner />
            <BrowserRouter>
              <RecoveryKeysGate />
              <Routes>
                <Route path="/welcome" element={<Landing />} />
                <Route path="/invite/:token" element={<InviteLanding />} />
                <Route element={<AppShell />}>
                  <Route path="/" element={<Dashboard />} />
                  <Route path="/services" element={<Services />} />
                  <Route path="/environments" element={<Environments />} />
                  <Route path="/jobs" element={<Jobs />} />
                  <Route path="/secrets" element={<Secrets />} />
                  <Route path="/audit" element={<AuditLog />} />
                  <Route path="/users" element={<Users />} />
                  <Route path="/profile" element={<Profile />} />
                  <Route path="/settings" element={<SettingsProfile />} />
                  <Route path="/auth-demo" element={<Suspense fallback={<div className="p-4 text-sm text-muted-foreground">Loading…</div>}><AuthFlows /></Suspense>} />
                </Route>
                {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
                <Route path="*" element={<NotFound />} />
              </Routes>
            </BrowserRouter>
        </AppStoreProvider>
      </TooltipProvider>
    </HelmetProvider>
  </QueryClientProvider>
);

export default App;
