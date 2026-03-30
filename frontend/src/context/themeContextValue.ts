import { createContext } from "react";
import type { ThemeType } from "../types/app";

export interface ThemeContextValue {
    theme: ThemeType;
    setTheme: (theme: ThemeType) => void;
    particlesEnabled: boolean;
    setParticlesEnabled: (enabled: boolean) => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);
