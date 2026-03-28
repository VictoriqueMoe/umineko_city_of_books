import {type PropsWithChildren, useCallback, useLayoutEffect, useState} from "react";
import type {ThemeType} from "../types/app";
import {ThemeContext} from "./themeContextValue";

const STORAGE_KEY = "ut-theme";
const PARTICLES_KEY = "ut-particles";
const DEFAULT_THEME: ThemeType = "featherine";

function getStoredTheme(): ThemeType {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored === "featherine" || stored === "bernkastel" || stored === "lambdadelta") {
            return stored;
        }
    } catch {
        // localStorage unavailable
    }
    return DEFAULT_THEME;
}

function getStoredParticles(): boolean {
    try {
        const stored = localStorage.getItem(PARTICLES_KEY);
        if (stored !== null) {
            return stored === "true";
        }
    } catch {
        // localStorage unavailable
    }
    return true;
}

export function ThemeProvider({ children }: PropsWithChildren) {
    const [theme, setThemeState] = useState<ThemeType>(getStoredTheme);
    const [particlesEnabled, setParticlesEnabledState] = useState(getStoredParticles);

    useLayoutEffect(() => {
        if (theme === DEFAULT_THEME) {
            document.documentElement.removeAttribute("data-theme");
        } else {
            document.documentElement.setAttribute("data-theme", theme);
        }
    }, [theme]);

    const setTheme = useCallback((newTheme: ThemeType) => {
        setThemeState(newTheme);
        try {
            localStorage.setItem(STORAGE_KEY, newTheme);
        } catch {
            // localStorage unavailable
        }
    }, []);

    const setParticlesEnabled = useCallback((enabled: boolean) => {
        setParticlesEnabledState(enabled);
        try {
            localStorage.setItem(PARTICLES_KEY, String(enabled));
        } catch {
            // localStorage unavailable
        }
    }, []);

    return (
        <ThemeContext.Provider value={{ theme, setTheme, particlesEnabled, setParticlesEnabled }}>
            {children}
        </ThemeContext.Provider>
    );
}
