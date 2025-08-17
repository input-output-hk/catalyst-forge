import { forge } from "@/lib/client";

// Thin wrappers around auth endpoints using the shared client

export const authApi = {
  async me() {
    return forge.raw.GET("/api/v1/auth/me");
  },
  async refresh() {
    return forge.raw.POST("/api/v1/auth/refresh", { body: {} as Record<string, never> });
  },
  async logout() {
    return forge.raw.POST("/api/v1/auth/logout", { body: {} as Record<string, never> });
  },
};



