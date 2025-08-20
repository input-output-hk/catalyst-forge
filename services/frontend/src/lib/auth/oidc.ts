import { Configuration, FrontendApi } from "@ory/client";

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

// Client for Kratos Public API routed via Caddy at /.ory/kratos/public
const kratos = new FrontendApi(
  new Configuration({ basePath: "/.ory/kratos/public", baseOptions: { withCredentials: true } })
);

/**
 * Start a browser login flow and submit provider=… using a real HTML form so the
 * browser naturally follows redirects (no CORS surprises).
 */
export async function loginWithProvider(provider: string, returnTo: string = "/"): Promise<void> {
  const { data } = await kratos.createBrowserLoginFlow();
  const flow = data as unknown as HasUi;

  const action = flow.ui?.action ?? `/.ory/kratos/public/self-service/login?flow=${flow.id ?? ""}`;
  const method = (flow.ui?.method ?? "POST").toUpperCase();
  const csrf = findNodeValue(flow.ui?.nodes ?? [], "csrf_token");

  const form = buildForm(action, method);
  appendHidden(form, "method", "oidc");
  appendHidden(form, "provider", provider);
  if (typeof csrf === "string" && csrf.length > 0) appendHidden(form, "csrf_token", csrf);
  // Note: returnTo is kept for parity; Kratos handles return_to internally on browser endpoints

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
  const { data } = await kratos.createBrowserRegistrationFlow();
  const flow = data as unknown as HasUi;

  const action =
    flow.ui?.action ?? `/.ory/kratos/public/self-service/registration?flow=${flow.id ?? ""}`;
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


