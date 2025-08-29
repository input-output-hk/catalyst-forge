import { Configuration, FrontendApi } from "@ory/client";
import { getKratosPublicBaseUrl } from "@/lib/auth/kratos";

// Minimal UI typing to avoid any-casts while keeping code clear
interface UiNodeAttributes {
  name?: string;
  value?: unknown;
}

interface UiNode {
  attributes?: UiNodeAttributes;
}

interface FlowUi {
  method?: string;
  action?: string;
  nodes?: UiNode[];
}

interface HasUi {
  ui?: FlowUi;
  id?: string;
}

// Client for Kratos Public API; base path is configurable at runtime
const kratosBase = getKratosPublicBaseUrl();
const kratos = new FrontendApi(
  new Configuration({ basePath: kratosBase, baseOptions: { withCredentials: true } })
);

/**
 * Start a browser login flow and submit provider=… using a real HTML form so the
 * browser naturally follows redirects (no CORS surprises).
 */
export async function loginWithProvider(provider: string, returnTo: string = "/"): Promise<void> {
  // Prefer reusing an existing Kratos login flow if the URL already contains ?flow=...
  // This preserves the original return_to (e.g., the Hydra login_challenge callback)
  console.log("STARTING LOGIN WITH PROVIDER", provider, returnTo);
  const url = new URL(window.location.href);
  const existingFlowId = url.searchParams.get("flow");
  console.log("existingFlowId", existingFlowId);

  // For custom UI: read the existing flow using a no-credentials JSON request.
  // Do NOT send cookies on this request or Kratos will require a CSRF header and return 403.
  async function readExistingFlow(flowId: string): Promise<HasUi | null> {
    try {
      const { data } = await kratos.getLoginFlow({ id: flowId });
      const f = data as HasUi;
      const hasAction = typeof f.ui?.action === "string" && f.ui.action.length > 0;
      const hasId = typeof f.id === "string" && f.id.length > 0;
      return (hasAction || hasId) ? f : null;
    } catch (err) {
      console.log("getLoginFlow error", err);
      return null;
    }
  }

  let flow: HasUi | null = null;

  // Try to load the existing flow first (if present in URL)
  if (existingFlowId) {
    try {
      console.log("readExistingFlow", existingFlowId);
      flow = await readExistingFlow(existingFlowId);
    } catch {
      console.log("readExistingFlow error", existingFlowId);
    }
  }

  // If no existing flow, initialize one via browser endpoint so Kratos sets cookies & returns here with ?flow=
  if (!flow) {
    const rt = returnTo && returnTo.length > 0 ? `?return_to=${encodeURIComponent(returnTo)}` : "";
    window.location.href = `${kratosBase}/self-service/login/browser${rt}`;
    return;
  }

  const action = flow.ui?.action ?? (flow.id ? `${kratosBase}/self-service/login?flow=${flow.id}` : "");
  if (!action) {
    const rt = returnTo && returnTo.length > 0 ? `?return_to=${encodeURIComponent(returnTo)}` : "";
    window.location.href = `${kratosBase}/self-service/login/browser${rt}`;
    return;
  }
  const method = (flow.ui?.method ?? "POST").toUpperCase();
  const csrf = findNodeValue(flow.ui?.nodes ?? [], "csrf_token");

  const form = buildForm(action, method);
  appendHidden(form, "method", "oidc");
  appendHidden(form, "provider", provider);
  if (typeof csrf === "string" && csrf.length > 0) appendHidden(form, "csrf_token", csrf);

  document.body.appendChild(form);
  form.submit();
}

// Backwards-compatible alias
export async function loginWithGithub(returnTo: string = "/"): Promise<void> {
  return loginWithProvider("github", returnTo);
}


/**
 * Start a browser registration flow and submit provider=… via HTML form.
 */
export async function registerWithProvider(provider: string, returnTo: string = "/"): Promise<void> {
  async function getFlowWithAction(): Promise<HasUi | null> {
    try {
      const { data } = await kratos.createBrowserRegistrationFlow();
      const f = data as unknown as HasUi;
      const hasAction = typeof f.ui?.action === "string" && f.ui.action.length > 0;
      const hasId = typeof f.id === "string" && f.id.length > 0;
      if (hasAction || hasId) return f;
      return null;
    } catch {
      return null;
    }
  }

  let flow = await getFlowWithAction();
  if (!flow) flow = await getFlowWithAction();

  if (!flow) {
    window.location.href = `${kratosBase}/self-service/registration/browser`;
    return;
  }

  const action = flow.ui?.action ?? (flow.id ? `${kratosBase}/self-service/registration?flow=${flow.id}` : "");
  if (!action) {
    window.location.href = `${kratosBase}/self-service/registration/browser`;
    return;
  }
  const method = (flow.ui?.method ?? "POST").toUpperCase();
  const csrf = findNodeValue(flow.ui?.nodes ?? [], "csrf_token");

  const form = buildForm(action, method);
  appendHidden(form, "method", "oidc");
  appendHidden(form, "provider", provider);
  if (typeof csrf === "string" && csrf.length > 0) appendHidden(form, "csrf_token", csrf);

  document.body.appendChild(form);
  form.submit();
}

// Backwards-compatible alias
export async function registerWithGithub(returnTo: string = "/"): Promise<void> {
  return registerWithProvider("github", returnTo);
}

// ---------- helpers ----------

function buildForm(action: string, method: string): HTMLFormElement {
  const form = document.createElement("form");
  form.method = method;
  form.action = action;
  return form;
}

function appendHidden(form: HTMLFormElement, name: string, value: string): void {
  const input = document.createElement("input");
  input.type = "hidden";
  input.name = name;
  input.value = value;
  form.appendChild(input);
}

function findNodeValue(nodes: UiNode[], attributeName: string): string | undefined {
  for (const node of nodes) {
    const name = node.attributes?.name ?? "";
    if (name === attributeName) {
      const raw = node.attributes?.value;
      return typeof raw === "string" ? raw : String(raw ?? "");
    }
  }
  return undefined;
}


