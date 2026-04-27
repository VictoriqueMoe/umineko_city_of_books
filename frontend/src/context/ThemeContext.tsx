import { type PropsWithChildren, useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FontType, ThemeType } from "../types/app";
import { useSiteInfo } from "../hooks/useSiteInfo";
import { useAuth } from "../hooks/useAuth";
import { useUpdateAppearance } from "../api/mutations/auth";
import { ThemeContext } from "./themeContextValue";

const STORAGE_KEY = "ut-theme";
const FONT_KEY = "ut-font";
const WIDE_LAYOUT_KEY = "ut-wide-layout";
const PARTICLES_KEY = "ut-particles";
const SECRETS_KEY = "ut-secrets";
const FALLBACK_THEME: ThemeType = "featherine";
const FALLBACK_FONT: FontType = "default";

const VALID_THEMES: Set<string> = new Set([
    "featherine",
    "bernkastel",
    "lambdadelta",
    "beatrice",
    "erika",
    "battler",
    "virgilia",
    "rika",
    "mion",
    "satoko",
    "miyao",
    "lingji",
    "stanislaw",
    "maria",
]);

const THEME_CSS_KEYS: Partial<Record<ThemeType, string>> = {
    maria: "_0x9e2a1c",
};

const THEME_REQUIRES_SECRET: Partial<Record<ThemeType, string>> = {
    maria: "witchHunter",
};

const VALID_FONTS: Set<string> = new Set(["default", "im-fell"]);

function isValidTheme(value: string): value is ThemeType {
    return VALID_THEMES.has(value);
}

function isValidFont(value: string): value is FontType {
    return VALID_FONTS.has(value);
}

function dataThemeAttr(t: ThemeType): string {
    return THEME_CSS_KEYS[t] ?? t;
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
    } catch {}
    return FALLBACK_THEME;
}

function getStoredFont(): FontType {
    try {
        const stored = localStorage.getItem(FONT_KEY);
        if (stored !== null && isValidFont(stored)) {
            return stored;
        }
    } catch {}
    return FALLBACK_FONT;
}

function getStoredParticles(): boolean {
    try {
        const stored = localStorage.getItem(PARTICLES_KEY);
        if (stored !== null) {
            return stored === "true";
        }
    } catch {}
    return true;
}

function getStoredWideLayout(): boolean {
    try {
        const stored = localStorage.getItem(WIDE_LAYOUT_KEY);
        if (stored !== null) {
            return stored === "true";
        }
    } catch {}
    return false;
}

function getStoredSecrets(): Set<string> {
    try {
        const raw = localStorage.getItem(SECRETS_KEY);
        if (raw) {
            const parsed = JSON.parse(raw);
            if (Array.isArray(parsed)) {
                return new Set(parsed.filter((v): v is string => typeof v === "string"));
            }
        }
    } catch {}
    return new Set();
}

function persistSecrets(secrets: Set<string>) {
    try {
        localStorage.setItem(SECRETS_KEY, JSON.stringify(Array.from(secrets)));
    } catch {}
}

export function ThemeProvider({ children }: PropsWithChildren) {
    const siteInfo = useSiteInfo();
    const { user } = useAuth();
    const [overrides, setOverrides] = useState<{
        userId: string | null;
        theme: ThemeType | null;
        font: FontType | null;
        wideLayout: boolean | null;
        secrets: Set<string> | null;
    }>(() => ({
        userId: null,
        theme: hasStoredTheme()
            ? getStoredTheme()
            : siteInfo.default_theme && isValidTheme(siteInfo.default_theme)
              ? siteInfo.default_theme
              : null,
        font: getStoredFont(),
        wideLayout: getStoredWideLayout(),
        secrets: getStoredSecrets(),
    }));
    const [particlesEnabled, setParticlesEnabledState] = useState(getStoredParticles);

    const activeUserId = user?.id ?? null;
    const activeOverrides = overrides.userId === activeUserId ? overrides : null;

    const fallbackTheme: ThemeType =
        siteInfo.default_theme && isValidTheme(siteInfo.default_theme) ? siteInfo.default_theme : FALLBACK_THEME;

    const userTheme = user?.theme && isValidTheme(user.theme) ? user.theme : null;
    const userFont = user?.font && isValidFont(user.font) ? user.font : null;
    const userWideLayout = typeof user?.wide_layout === "boolean" ? user.wide_layout : null;
    const userSecrets = user && Array.isArray(user.secrets) ? new Set<string>(user.secrets) : null;

    const storedTheme = hasStoredTheme() ? getStoredTheme() : null;

    let theme: ThemeType = activeOverrides?.theme ?? userTheme ?? storedTheme ?? fallbackTheme;
    const font: FontType = activeOverrides?.font ?? userFont ?? getStoredFont();
    const wideLayout: boolean = activeOverrides?.wideLayout ?? userWideLayout ?? getStoredWideLayout();
    const secrets: Set<string> = activeOverrides?.secrets ?? userSecrets ?? getStoredSecrets();

    const requiredSecret = THEME_REQUIRES_SECRET[theme];
    if (requiredSecret && !secrets.has(requiredSecret)) {
        theme = fallbackTheme;
    }

    const secretsRef = useRef<Set<string>>(secrets);
    useEffect(() => {
        secretsRef.current = secrets;
    }, [secrets]);

    useEffect(() => {
        try {
            localStorage.setItem(STORAGE_KEY, theme);
        } catch {}
    }, [theme]);

    useEffect(() => {
        try {
            localStorage.setItem(FONT_KEY, font);
        } catch {}
    }, [font]);

    useEffect(() => {
        try {
            localStorage.setItem(WIDE_LAYOUT_KEY, String(wideLayout));
        } catch {}
    }, [wideLayout]);

    useEffect(() => {
        persistSecrets(secrets);
    }, [secrets]);

    const hasSecret = useCallback((id: string) => secrets.has(id), [secrets]);

    const patchOverrides = useCallback(
        (update: { theme?: ThemeType; font?: FontType; wideLayout?: boolean; secrets?: Set<string> }) => {
            setOverrides(prev => {
                const base =
                    prev.userId === activeUserId
                        ? prev
                        : {
                              userId: activeUserId,
                              theme: null,
                              font: null,
                              wideLayout: null,
                              secrets: null,
                          };
                return {
                    userId: activeUserId,
                    theme: update.theme ?? base.theme,
                    font: update.font ?? base.font,
                    wideLayout: update.wideLayout ?? base.wideLayout,
                    secrets: update.secrets ?? base.secrets,
                };
            });
        },
        [activeUserId],
    );

    useLayoutEffect(() => {
        if (theme === FALLBACK_THEME) {
            document.documentElement.removeAttribute("data-theme");
        } else {
            document.documentElement.setAttribute("data-theme", dataThemeAttr(theme));
        }
    }, [theme]);

    useLayoutEffect(() => {
        if (font === FALLBACK_FONT) {
            document.documentElement.removeAttribute("data-font");
        } else {
            document.documentElement.setAttribute("data-font", font);
        }
    }, [font]);

    useLayoutEffect(() => {
        if (wideLayout) {
            document.documentElement.setAttribute("data-width", "wide");
        } else {
            document.documentElement.removeAttribute("data-width");
        }
    }, [wideLayout]);

    const updateAppearanceMutation = useUpdateAppearance();
    const persistAppearance = useCallback(
        (nextTheme: ThemeType, nextFont: FontType, nextWide: boolean) => {
            if (!user) {
                return;
            }
            updateAppearanceMutation.mutate({ theme: nextTheme, font: nextFont, wideLayout: nextWide });
        },
        [user, updateAppearanceMutation],
    );

    const setTheme = useCallback(
        (newTheme: ThemeType) => {
            patchOverrides({ theme: newTheme });
            persistAppearance(newTheme, font, wideLayout);
        },
        [font, wideLayout, persistAppearance, patchOverrides],
    );

    const setFont = useCallback(
        (newFont: FontType) => {
            patchOverrides({ font: newFont });
            persistAppearance(theme, newFont, wideLayout);
        },
        [theme, wideLayout, persistAppearance, patchOverrides],
    );

    const setWideLayout = useCallback(
        (enabled: boolean) => {
            patchOverrides({ wideLayout: enabled });
            persistAppearance(theme, font, enabled);
        },
        [theme, font, persistAppearance, patchOverrides],
    );

    const setParticlesEnabled = useCallback((enabled: boolean) => {
        setParticlesEnabledState(enabled);
        try {
            localStorage.setItem(PARTICLES_KEY, String(enabled));
        } catch {}
    }, []);

    const addSecret = useCallback(
        (id: string) => {
            if (secretsRef.current.has(id)) {
                return;
            }
            const next = new Set(secretsRef.current);
            next.add(id);
            secretsRef.current = next;
            patchOverrides({ secrets: next });
        },
        [patchOverrides],
    );

    return (
        <ThemeContext.Provider
            value={{
                theme,
                setTheme,
                font,
                setFont,
                wideLayout,
                setWideLayout,
                particlesEnabled,
                setParticlesEnabled,
                hasSecret,
                addSecret,
            }}
        >
            {children}
        </ThemeContext.Provider>
    );
}
