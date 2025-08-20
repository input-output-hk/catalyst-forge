import { Configuration, FrontendApi } from "@ory/client";

interface VerifiableAddress {
  value?: string;
  verified?: boolean;
}

interface IdentityLike {
  traits?: { email?: string };
  verifiable_addresses?: VerifiableAddress[];
}

const kratos = new FrontendApi(
  new Configuration({ basePath: "/.ory/kratos/public", baseOptions: { withCredentials: true } })
);

/**
 * Send an email verification link when the session identity has an unverified email.
 * Safe to call opportunistically; exits quickly if already verified or data missing.
 */
export async function sendEmailVerificationIfNeeded(identity: unknown): Promise<void> {
  const typed = identity as IdentityLike | null;
  const email = typed?.traits?.email ?? undefined;
  const addresses = typed?.verifiable_addresses ?? [];
  if (!email || addresses.length === 0) return;

  const match = addresses.find((a) => (a?.value ?? "").toLowerCase() === email.toLowerCase());
  if (!match || match.verified === true) return;

  const { data } = await kratos.createBrowserVerificationFlow();
  const flowId = (data as unknown as { id?: string })?.id ?? "";
  await kratos.updateVerificationFlow({
    flow: flowId,
    updateVerificationFlowBody: { method: "link", email },
  });
}


