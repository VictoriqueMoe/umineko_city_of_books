import type { ThemeType } from "../types/app";

const LIGHT_THEMES: Set<string> = new Set<ThemeType>(["virgilia"]);

export function isLightTheme(theme: string | undefined | null): boolean {
    if (!theme) {
        return false;
    }

    return LIGHT_THEMES.has(theme);
}
