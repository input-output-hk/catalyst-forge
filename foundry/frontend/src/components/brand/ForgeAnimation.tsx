import { cn } from "@/lib/utils";

interface ForgeAnimationProps {
  className?: string;
}

export function ForgeAnimation({ className }: ForgeAnimationProps) {
  return (
    <div className={cn("forge-background", className)}>
      {/* Background forge glow */}
      <div className="absolute inset-0 bg-gradient-radial from-amber-500/10 via-orange-500/5 to-transparent" />
      
      {/* Animated SVG forge scene */}
      <svg
        viewBox="0 0 800 600"
        className="absolute inset-0 w-full h-full"
        preserveAspectRatio="xMidYMid slice"
      >
        {/* Forge base structure */}
        <defs>
          <radialGradient id="forgeGlow" cx="50%" cy="70%" r="40%">
            <stop offset="0%" stopColor="hsl(var(--amber-400))" stopOpacity="0.3" />
            <stop offset="70%" stopColor="hsl(var(--orange-500))" stopOpacity="0.1" />
            <stop offset="100%" stopColor="transparent" />
          </radialGradient>
          
          <filter id="emberGlow">
            <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
            <feMerge> 
              <feMergeNode in="coloredBlur"/>
              <feMergeNode in="SourceGraphic"/>
            </feMerge>
          </filter>
        </defs>

        {/* Forge background glow */}
        <ellipse 
          cx="400" 
          cy="450" 
          rx="200" 
          ry="100" 
          fill="url(#forgeGlow)"
          className="forge-ember"
        />

        {/* Anvil */}
        <g className="anvil-glow" style={{ transformOrigin: '400px 420px' }}>
          {/* Anvil body */}
          <rect x="350" y="400" width="100" height="40" rx="5" fill="hsl(var(--steel-600))" />
          <rect x="340" y="440" width="120" height="20" rx="10" fill="hsl(var(--steel-700))" />
          
          {/* Anvil horn */}
          <path 
            d="M 450 400 L 480 390 L 485 410 L 450 420 Z" 
            fill="hsl(var(--steel-500))"
          />
        </g>

        {/* Hammer */}
        <g className="hammer-animation" style={{ transformOrigin: '500px 380px' }}>
          {/* Hammer handle */}
          <rect x="495" y="300" width="8" height="80" rx="4" fill="hsl(var(--amber-800))" />
          
          {/* Hammer head */}
          <rect x="485" y="280" width="28" height="25" rx="3" fill="hsl(var(--steel-400))" />
          <rect x="487" y="282" width="24" height="6" rx="1" fill="hsl(var(--steel-300))" />
        </g>

        {/* Sparks (multiple particles with staggered delays) */}
        <g>
          {[...Array(6)].map((_, i) => (
            <circle
              key={i}
              cx={420 + i * 8}
              cy={410}
              r="1.5"
              fill="hsl(var(--amber-300))"
              className="spark-particle"
              style={{ 
                animationDelay: `${i * 0.2}s`,
                filter: 'url(#emberGlow)'
              }}
            />
          ))}
        </g>

        {/* Floating embers */}
        <g>
          {[...Array(4)].map((_, i) => (
            <circle
              key={`ember-${i}`}
              cx={380 + i * 30}
              cy={380 - i * 10}
              r="2"
              fill="hsl(var(--orange-400))"
              className="forge-ember"
              style={{ 
                animationDelay: `${i * 0.8}s`,
                filter: 'url(#emberGlow)',
                opacity: 0.6
              }}
            />
          ))}
        </g>
      </svg>
    </div>
  );
}