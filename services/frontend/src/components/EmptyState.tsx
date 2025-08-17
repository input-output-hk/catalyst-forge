import React from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type CTA = {
  label: string;
  onClick: () => void;
};

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  body?: string;
  primaryCta?: CTA;
  secondaryCta?: CTA;
  className?: string;
}

export default function EmptyState({
  icon,
  title,
  body,
  primaryCta,
  secondaryCta,
  className,
}: EmptyStateProps) {
  return (
    <section
      className={cn(
        "rounded-lg border bg-foreground/[0.02] p-10 text-center text-muted-foreground",
        className
      )}
      aria-live="polite"
    >
      {icon ? <div className="opacity-40 mb-3">{icon}</div> : null}
      <h2 className="text-lg font-medium text-foreground">{title}</h2>
      {body ? <p className="mt-1">{body}</p> : null}
      {(primaryCta || secondaryCta) && (
        <div className="mt-3 flex items-center justify-center gap-2">
          {primaryCta ? (
            <Button variant="hero" onClick={primaryCta.onClick}>
              {primaryCta.label}
            </Button>
          ) : null}
          {secondaryCta ? (
            <Button variant="outline" onClick={secondaryCta.onClick}>
              {secondaryCta.label}
            </Button>
          ) : null}
        </div>
      )}
    </section>
  );
}
