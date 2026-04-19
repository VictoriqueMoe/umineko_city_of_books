import { type PropsWithChildren, useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FontType, ThemeType } from "../types/app";
import { useSiteInfo } from "../hooks/useSiteInfo";
import { useAuth } from "../hooks/useAuth";
import { updateAppearance } from "../api/endpoints";
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
    const [theme, setThemeState] = useState<ThemeType>(() => {
        if (hasStoredTheme()) {
            return getStoredTheme();
        }
        if (siteInfo.default_theme && isValidTheme(siteInfo.default_theme)) {
            return siteInfo.default_theme;
        }
        return FALLBACK_THEME;
    });
    const [font, setFontState] = useState<FontType>(getStoredFont);
    const [wideLayout, setWideLayoutState] = useState(getStoredWideLayout);
    const [particlesEnabled, setParticlesEnabledState] = useState(getStoredParticles);
    const [secrets, setSecretsState] = useState<Set<string>>(getStoredSecrets);
    const hydratedUserRef = useRef<string | null>(null);
    const secretsRef = useRef<Set<string>>(secrets);

    useEffect(() => {
        secretsRef.current = secrets;
    }, [secrets]);

    const hasSecret = useCallback((id: string) => secrets.has(id), [secrets]);

    useEffect(() => {
        if (!user) {
            hydratedUserRef.current = null;
            return;
        }
        if (hydratedUserRef.current === user.id) {
            return;
        }
        hydratedUserRef.current = user.id;
        if (user.theme && isValidTheme(user.theme)) {
            // eslint-disable-next-line react-hooks/set-state-in-effect
            setThemeState(user.theme);
            try {
                localStorage.setItem(STORAGE_KEY, user.theme);
            } catch {}
        }
        if (user.font && isValidFont(user.font)) {
            setFontState(user.font);
            try {
                localStorage.setItem(FONT_KEY, user.font);
            } catch {}
        }
        if (typeof user.wide_layout === "boolean") {
            setWideLayoutState(user.wide_layout);
            try {
                localStorage.setItem(WIDE_LAYOUT_KEY, String(user.wide_layout));
            } catch {}
        }
        if (Array.isArray(user.secrets) && user.secrets.length > 0) {
            const next = new Set(secretsRef.current);
            for (const id of user.secrets) {
                next.add(id);
            }
            secretsRef.current = next;
            setSecretsState(next);
            persistSecrets(next);
        }
    }, [user]);

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

    const persistAppearance = useCallback(
        (nextTheme: ThemeType, nextFont: FontType, nextWide: boolean) => {
            if (!user) {
                return;
            }
            updateAppearance(nextTheme, nextFont, nextWide).catch(() => {});
        },
        [user],
    );

    const setTheme = useCallback(
        (newTheme: ThemeType) => {
            setThemeState(newTheme);
            try {
                localStorage.setItem(STORAGE_KEY, newTheme);
            } catch {}
            persistAppearance(newTheme, font, wideLayout);
        },
        [font, wideLayout, persistAppearance],
    );

    const setFont = useCallback(
        (newFont: FontType) => {
            setFontState(newFont);
            try {
                localStorage.setItem(FONT_KEY, newFont);
            } catch {}
            persistAppearance(theme, newFont, wideLayout);
        },
        [theme, wideLayout, persistAppearance],
    );

    const setWideLayout = useCallback(
        (enabled: boolean) => {
            setWideLayoutState(enabled);
            try {
                localStorage.setItem(WIDE_LAYOUT_KEY, String(enabled));
            } catch {}
            persistAppearance(theme, font, enabled);
        },
        [theme, font, persistAppearance],
    );

    const setParticlesEnabled = useCallback((enabled: boolean) => {
        setParticlesEnabledState(enabled);
        try {
            localStorage.setItem(PARTICLES_KEY, String(enabled));
        } catch {}
    }, []);

    const addSecret = useCallback((id: string) => {
        if (secretsRef.current.has(id)) {
            return;
        }
        const next = new Set(secretsRef.current);
        next.add(id);
        secretsRef.current = next;
        setSecretsState(next);
        persistSecrets(next);
    }, []);

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
