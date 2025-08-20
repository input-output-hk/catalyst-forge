import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { HelmetProvider } from "react-helmet-async";
import { AppStoreProvider } from "@/store/app-store";
import { AppBootstrap } from "@/components/AppBootstrap";
import Dashboard from "@/pages/Dashboard";
import Services from "@/pages/Services";
import Environments from "@/pages/Environments";
import Jobs from "@/pages/Jobs";
import Secrets from "@/pages/Secrets";
import AuditLog from "@/pages/AuditLog";
import SettingsProfile from "@/pages/SettingsProfile";
import Profile from "@/pages/Profile";
import Users from "@/pages/Users";
import NotFound from "./pages/NotFound";
import { AppShell } from "@/components/layout/AppShell";
import Landing from "@/pages/Landing";
import RequireAuth, { RequireAdmin } from "@/components/auth/RequireAuth";
import KratosLogout from "@/pages/kratos/Logout";
import KratosError from "@/pages/kratos/Error";

const queryClient = new QueryClient();

const App = () => (
  <QueryClientProvider client={queryClient}>
    <HelmetProvider>
      <TooltipProvider>
        <AppStoreProvider>
          <Toaster />
          <Sonner />
          <BrowserRouter>
            <AppBootstrap />
            <Routes>
              <Route path="/welcome" element={<Landing />} />
              <Route path="/auth/logout" element={<KratosLogout />} />
              <Route path="/auth/error" element={<KratosError />} />
              <Route element={<RequireAuth />}>
                <Route element={<AppShell />}>
                  <Route path="/" element={<Dashboard />} />
                  <Route path="/services" element={<Services />} />
                  <Route path="/environments" element={<Environments />} />
                  <Route path="/jobs" element={<Jobs />} />
                  <Route path="/secrets" element={<Secrets />} />
                  <Route
                    path="/audit"
                    element={
                      <RequireAdmin>
                        <AuditLog />
                      </RequireAdmin>
                    }
                  />
                  <Route
                    path="/users"
                    element={
                      <RequireAdmin>
                        <Users />
                      </RequireAdmin>
                    }
                  />
                  <Route path="/profile" element={<Profile />} />
                  <Route path="/settings" element={<SettingsProfile />} />
                </Route>
              </Route>
              <Route path="*" element={<NotFound />} />
            </Routes>
          </BrowserRouter>
        </AppStoreProvider>
      </TooltipProvider>
    </HelmetProvider>
  </QueryClientProvider>
);

export default App;
