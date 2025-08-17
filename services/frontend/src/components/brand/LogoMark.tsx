import * as React from "react";
import { BRAND } from "@/lib/brand";

export type LogoMarkProps = {
  size?: number;
  className?: string;
  rounded?: boolean;
};

export function LogoMark({ size = 24, className = "", rounded = false }: LogoMarkProps) {
  const [fallback, setFallback] = React.useState(false);

  const isPlaceholder = !BRAND.logoColor || BRAND.logoColor.includes("placeholder.svg");

  if (fallback || isPlaceholder) {
    return (
      <div
        className={`inline-flex items-center justify-center ${rounded ? "rounded-full" : "rounded-sm"} ${className}`}
        style={{ width: size, height: size }}
        aria-label="Catalyst Forge"
      >
        <span className="sr-only">Catalyst Forge</span>
        {/* Fallback: brand gradient block to ensure visibility in both themes */}
        <div
          className={`h-full w-full ${rounded ? "rounded-full" : "rounded-sm"}`}
          style={{ backgroundImage: "var(--gradient-primary)" }}
          aria-hidden
        />
      </div>
    );
  }

  return (
    <div className={`relative ${className}`} style={{ width: size, height: size }}>
      {/* Light mode: colored logo */}
      <img
        src={BRAND.logoColor}
        alt="Catalyst Forge logo"
        className="absolute inset-0 h-full w-full object-contain dark:hidden"
        onError={() => setFallback(true)}
        loading="lazy"
      />
      {/* Dark mode: monochrome (white) logo */}
      {/* Dark mode: masked white logo using inverted mono asset */}
      <div
        role="img"
        aria-label="Catalyst Forge logo"
        className="absolute inset-0 hidden dark:block"
        style={{
          backgroundColor: "hsl(var(--foreground))",
          WebkitMaskImage: `url(${BRAND.logoMono})`,
          maskImage: `url(${BRAND.logoMono})`,
          WebkitMaskRepeat: "no-repeat",
          maskRepeat: "no-repeat",
          WebkitMaskPosition: "center",
          maskPosition: "center",
          WebkitMaskSize: "contain",
          maskSize: "contain",
        }}
      />
      {/* Preload invisible img to preserve onError fallback handling */}
      <img
        src={BRAND.logoMono}
        alt=""
        className="hidden"
        onError={() => setFallback(true)}
        loading="eager"
        aria-hidden
      />
    </div>
  );
}
