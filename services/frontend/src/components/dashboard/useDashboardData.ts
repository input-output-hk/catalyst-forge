import { useEffect, useState } from "react";
import { getKpis, getEnvironments, getActivity, type Kpi, type EnvHealth, type ActivityItem } from "@/lib/mock/dashboard";
import { withLatency } from "@/mocks/latency";

export function useDashboardData() {
  const [kpis, setKpis] = useState<Kpi[] | null>(null);
  const [envs, setEnvs] = useState<EnvHealth[] | null>(null);
  const [activity, setActivity] = useState<ActivityItem[] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let timer: number | undefined;
    let mounted = true;

    async function load() {
      setLoading(true);
      await withLatency(250, 600);
      if (!mounted) return;
      setKpis(getKpis());
      setEnvs(getEnvironments());
      setActivity(getActivity());
      setLoading(false);
    }

    load();
    // refresh every 15s
    timer = window.setInterval(() => {
      setKpis(getKpis());
      setEnvs(getEnvironments());
      setActivity(getActivity());
    }, 15000);

    return () => {
      mounted = false;
      if (timer) window.clearInterval(timer);
    };
  }, []);

  return { kpis, envs, activity, loading };
}
