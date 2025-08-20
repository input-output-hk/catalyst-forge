import { useEffect, useMemo, useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useAppStore } from "@/store/app-store";
/* eslint-disable @typescript-eslint/no-explicit-any */
import { forge } from "@/lib/client";
import type { paths } from "forge-client";
// legacy webauthn helpers removed
import { extractErrorMessage, readResponseError, refreshAccessToken } from "@/lib/api";

// Type definitions for API endpoints
type BootstrapRequestBody =
  paths["/api/v1/auth/bootstrap"]["post"]["requestBody"]["content"]["application/json"];
type BeginCredentialRequestBody = never;
type BeginCredentialResponse = never;
type CompleteCredentialRequestBody = never;
type MeResponse = paths["/api/v1/auth/me"]["get"]["responses"][200]["content"]["application/json"];

// WebAuthn response shape from server
interface WebAuthnServerResponse { }

// legacy removed

const schema = z.object({
  email: z.string().email(),
  bootstrap_token: z.string().min(32).max(64),
  device_name: z.string().min(1),
});

export default function Bootstrap() {
  const [params] = useSearchParams();
  const nav = useNavigate();
  const { actions } = useAppStore();
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: params.get("email") ?? "",
      bootstrap_token: params.get("token") ?? "",
      device_name: "Admin Key",
    },
  });

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Note: do not refresh on mount; bootstrap now sets CSRF cookie, and we refresh only if needed after bootstrap

  const onSubmit = async (values: z.infer<typeof schema>) => {
    setSubmitting(true);
    setError(null);

    try {
      // Step 1: Bootstrap the authentication process
      const bootstrapBody: BootstrapRequestBody = {
        email: values.email,
        bootstrap_token: values.bootstrap_token,
      };

      const bootstrapResponse = await forge.raw.POST("/api/v1/auth/bootstrap", {
        body: bootstrapBody,
      });

      if (!bootstrapResponse.response.ok) {
        throw new Error("bootstrap failed");
      }

      // Ensure access token is available before subsequent POSTs to avoid auth retry/replay
      try {
        const token = await refreshAccessToken();
        if (
          token &&
          (forge as unknown as { setAccessToken?: (t: string) => void }).setAccessToken
        ) {
          (forge as unknown as { setAccessToken?: (t: string) => void }).setAccessToken?.(token);
        }
      } catch {
        /* ignore */
      }

      // Step 2 removed: legacy WebAuthn registration

      // Step 3: Fetch user profile to initialize UI session
      const profileResponse = await forge.raw.GET("/api/v1/auth/me");

      if (profileResponse.response.ok && profileResponse.data) {
        const profileData = profileResponse.data as MeResponse;

        // Extract email from response (try multiple paths for compatibility)
        const userEmail = profileData?.email || (profileData as any)?.user?.email || "user";

        actions.login(userEmail, []);
      }

      // Navigate to home page
      nav("/", { replace: true });
    } catch (error: unknown) {
      const errorMessage = extractErrorMessage(error, "Unknown error");
      setError(errorMessage);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="container py-8 max-w-xl">
      <Card>
        <CardHeader>
          <CardTitle>Admin Bootstrap</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4" noValidate>
              <FormField
                name="email"
                control={form.control}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input type="email" placeholder="admin@yourcompany.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                name="bootstrap_token"
                control={form.control}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Bootstrap token</FormLabel>
                    <FormControl>
                      <Input placeholder="paste token" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                name="device_name"
                control={form.control}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Device name</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g., MacBook Pro (Chrome)" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {error && <div className="text-sm text-red-600">{error}</div>}
              <Button type="submit" disabled={submitting}>
                {submitting ? "Setting up…" : "Bootstrap"}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>
    </section>
  );
}
