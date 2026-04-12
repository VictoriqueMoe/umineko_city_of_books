import { createContext } from "react";
import type { FontType, ThemeType } from "../types/app";

export interface ThemeContextValue {
    theme: ThemeType;
    setTheme: (theme: ThemeType) => void;
    font: FontType;
    setFont: (font: FontType) => void;
    wideLayout: boolean;
    setWideLayout: (enabled: boolean) => void;
    particlesEnabled: boolean;
    setParticlesEnabled: (enabled: boolean) => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);
