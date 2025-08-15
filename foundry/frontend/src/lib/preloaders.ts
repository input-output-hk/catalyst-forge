// Preloaders for route-based code-split chunks
// Ensures each chunk is only requested once per session

export const preloadAuthFlows: () => Promise<any> = (() => {
  let promise: Promise<any> | null = null;
  return () => (promise ??= import("@/pages/AuthFlows"));
})();
