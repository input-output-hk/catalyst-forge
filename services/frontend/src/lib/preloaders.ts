// Preloaders for route-based code-split chunks
// Ensures each chunk is only requested once per session

export const preloadAuthFlows: () => Promise<typeof import("@/pages/AuthFlows")> = (() => {
  let promise: Promise<typeof import("@/pages/AuthFlows")> | null = null;
  return () => (promise ??= import("@/pages/AuthFlows"));
})();
