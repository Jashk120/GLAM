"use client"
import { ReactLenis } from "lenis/react";
import { ReactNode } from "react";
export default function SmoothScrollProvider({children }: { childern : ReactNode  }){
  return (
    <ReactLenis root options={{ lerp : 0.1,duration : 1.2 }}>
      {childern}
    </ReactLenis>
  );
}

