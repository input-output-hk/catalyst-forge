import { Outlet } from "react-router-dom";
import { AppLayout } from "./AppLayout";

export const AppShell = () => {
  return (
    <AppLayout>
      <Outlet />
    </AppLayout>
  );
};
