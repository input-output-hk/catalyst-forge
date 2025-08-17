import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Download, Pause, Play } from "lucide-react";
import { useAppStore } from "@/store/app-store";
import type { JobRun } from "@/mocks/fixtures";

export const LogViewer = ({ run }: { run: JobRun }) => {
  const { state } = useAppStore();
  const [lines, setLines] = useState<string[]>(run.logs ?? []);
  const [paused, setPaused] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => setLines(run.logs ?? []), [run.id]);

  useEffect(() => {
    if (!state.flags.streamLogs || paused || run.status !== "running") return;
    const id = setInterval(() => {
      setLines((ls) => [...ls, `log ${ls.length + 1}: working…`]);
      ref.current?.scrollTo({ top: ref.current.scrollHeight });
    }, 700);
    return () => clearInterval(id);
  }, [state.flags.streamLogs, paused, run.status]);

  const download = () => {
    const blob = new Blob([lines.join("\n")], { type: "text/plain" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `${run.id}.log`;
    a.click();
  };

  return (
    <div className="flex flex-col h-[420px]">
      <div className="mb-2 flex items-center gap-2">
        <Button size="sm" variant="secondary" onClick={() => setPaused((p) => !p)}>
          {paused ? <Play className="mr-2 h-4 w-4" /> : <Pause className="mr-2 h-4 w-4" />}
          {paused ? "Resume" : "Pause"}
        </Button>
        <Button size="sm" variant="outline" onClick={download}>
          <Download className="mr-2 h-4 w-4" /> Download
        </Button>
      </div>
      <div
        ref={ref}
        className="flex-1 overflow-auto rounded-md border bg-secondary/20 p-3 font-mono text-xs leading-relaxed"
      >
        {lines.length === 0 ? (
          <div className="text-muted-foreground">No logs yet…</div>
        ) : (
          lines.map((l, i) => <div key={i}>{l}</div>)
        )}
      </div>
    </div>
  );
};
