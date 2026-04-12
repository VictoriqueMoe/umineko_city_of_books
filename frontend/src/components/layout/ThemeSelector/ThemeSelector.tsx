import { useCallback, useRef, useState } from "react";
import { useTheme } from "../../../hooks/useTheme";
import { useClickOutside } from "../../../hooks/useClickOutside";
import type { FontType, ThemeType } from "../../../types/app";
import { ToggleSwitch } from "../../ToggleSwitch/ToggleSwitch";
import styles from "./ThemeSelector.module.css";

interface ThemeDefinition {
    id: ThemeType;
    name: string;
    description: string;
}

interface FontDefinition {
    id: FontType;
    name: string;
    description: string;
    previewClass: string;
}

const THEMES: ThemeDefinition[] = [
    { id: "featherine", name: "Featherine", description: "Witch of Theatergoing, Drama, and Spectating" },
    { id: "beatrice", name: "Beatrice", description: "The Golden and Endless Witch" },
    { id: "bernkastel", name: "Lady Bernkastel", description: "Witch of Miracles" },
    { id: "lambdadelta", name: "Lady Lambdadelta", description: "Witch of Certainty" },
    { id: "erika", name: "Erika Furudo", description: "The Witch of Truth" },
    { id: "battler", name: "Battler Ushiromiya", description: "The Endless Sorcerer" },
    { id: "rika", name: "Rika Furude", description: "Nipah~!" },
    { id: "mion", name: "Mion Sonozaki", description: "The Club President" },
    { id: "satoko", name: "Satoko Houjou", description: "The Trap Master" },
];

const FONTS: FontDefinition[] = [
    {
        id: "default",
        name: "Cinzel & Garamond",
        description: "The classic look",
        previewClass: styles.fontPreviewDefault,
    },
    {
        id: "im-fell",
        name: "IM Fell English",
        description: "Antique grimoire print",
        previewClass: styles.fontPreviewImFell,
    },
];

export function ThemeSelector() {
    const { theme, setTheme, font, setFont, wideLayout, setWideLayout, particlesEnabled, setParticlesEnabled } =
        useTheme();
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    const currentTheme = THEMES.find(t => t.id === theme);
    useClickOutside(
        dropdownRef,
        useCallback(() => setIsOpen(false), []),
    );

    return (
        <div className={styles.selector} ref={dropdownRef}>
            <button
                className={styles.trigger}
                onClick={() => setIsOpen(!isOpen)}
                aria-expanded={isOpen}
                aria-haspopup="listbox"
            >
                <span className={styles.triggerLabel}>Theme</span>
                <span className={styles.triggerSep}>{"\u2726"}</span>
                <span className={styles.triggerName}>{currentTheme?.name}</span>
                <span className={`${styles.chevron}${isOpen ? ` ${styles.chevronOpen}` : ""}`}>{"\u25BC"}</span>
            </button>

            {isOpen && (
                <div className={styles.dropdown} role="listbox">
                    {THEMES.map(t => (
                        <button
                            key={t.id}
                            className={`${styles.option}${t.id === theme ? ` ${styles.optionActive}` : ""}`}
                            onClick={() => {
                                setTheme(t.id);
                                setIsOpen(false);
                            }}
                            role="option"
                            aria-selected={t.id === theme}
                        >
                            <div className={styles.optionInfo}>
                                <span className={styles.optionName}>{t.name}</span>
                                <span className={styles.optionDesc}>{t.description}</span>
                            </div>
                            {t.id === theme && <span className={styles.check}>{"\u2713"}</span>}
                        </button>
                    ))}
                    <div className={styles.divider} />
                    <span className={styles.sectionLabel}>Font</span>
                    {FONTS.map(f => (
                        <button
                            key={f.id}
                            className={`${styles.option}${f.id === font ? ` ${styles.optionActive}` : ""}`}
                            onClick={() => {
                                setFont(f.id);
                                setIsOpen(false);
                            }}
                            role="option"
                            aria-selected={f.id === font}
                        >
                            <div className={styles.optionInfo}>
                                <span className={`${styles.optionName} ${f.previewClass}`}>{f.name}</span>
                                <span className={styles.optionDesc}>{f.description}</span>
                            </div>
                            {f.id === font && <span className={styles.check}>{"\u2713"}</span>}
                        </button>
                    ))}
                    <div className={styles.divider} />
                    <ToggleSwitch
                        enabled={wideLayout}
                        onChange={setWideLayout}
                        label="Wide layout"
                        description="Use the full width of the screen"
                    />
                    <ToggleSwitch
                        enabled={particlesEnabled}
                        onChange={setParticlesEnabled}
                        label="Particles"
                        description="Floating butterflies & sparkles"
                    />
                </div>
            )}
        </div>
    );
}
