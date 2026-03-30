import {type PropsWithChildren, useCallback, useEffect, useLayoutEffect, useState} from "react";
import type {ThemeType} from "../types/app";
import {getSiteInfo} from "../api/endpoints";
import {ThemeContext} from "./themeContextValue";

const STORAGE_KEY = "ut-theme";
const PARTICLES_KEY = "ut-particles";
const FALLBACK_THEME: ThemeType = "featherine";

function isValidTheme(value: string): value is ThemeType {
    return value === "featherine" || value === "bernkastel" || value === "lambdadelta";
}

function hasStoredTheme(): boolean {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        return stored !== null && isValidTheme(stored);
    } catch {
        return false;
    }
}

function getStoredTheme(): ThemeType {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored !== null && isValidTheme(stored)) {
            return stored;
        }
    } catch {
        void 0;
    }
    return FALLBACK_THEME;
}

function getStoredParticles(): boolean {
    try {
        const stored = localStorage.getItem(PARTICLES_KEY);
        if (stored !== null) {
            return stored === "true";
        }
    } catch {
        void 0;
    }
    return true;
}

export function ThemeProvider({ children }: PropsWithChildren) {
    const [theme, setThemeState] = useState<ThemeType>(getStoredTheme);
    const [particlesEnabled, setParticlesEnabledState] = useState(getStoredParticles);

    useEffect(() => {
        if (hasStoredTheme()) {
            return;
        }
        getSiteInfo()
            .then(info => {
                if (info.default_theme && isValidTheme(info.default_theme)) {
                    setThemeState(info.default_theme);
                }
            })
            .catch(() => {});
    }, []);

    useLayoutEffect(() => {
        if (theme === FALLBACK_THEME) {
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
            void 0;
        }
    }, []);

    const setParticlesEnabled = useCallback((enabled: boolean) => {
        setParticlesEnabledState(enabled);
        try {
            localStorage.setItem(PARTICLES_KEY, String(enabled));
        } catch {
            void 0;
        }
    }, []);

    return (
        <ThemeContext.Provider value={{ theme, setTheme, particlesEnabled, setParticlesEnabled }}>
            {children}
        </ThemeContext.Provider>
    );
}
