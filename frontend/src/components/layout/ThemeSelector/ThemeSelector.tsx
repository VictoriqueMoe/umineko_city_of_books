import {useCallback, useRef, useState} from "react";
import {useTheme} from "../../../hooks/useTheme";
import {useClickOutside} from "../../../hooks/useClickOutside";
import type {ThemeType} from "../../../types/app";
import {ToggleSwitch} from "../../ToggleSwitch/ToggleSwitch";
import styles from "./ThemeSelector.module.css";

interface ThemeDefinition {
    id: ThemeType;
    name: string;
    description: string;
}

const THEMES: ThemeDefinition[] = [
    { id: "featherine", name: "Featherine", description: "Witch of Theatergoing, Drama, and Spectating" },
    { id: "bernkastel", name: "Lady Bernkastel", description: "Witch of Miracles" },
    { id: "lambdadelta", name: "Lady Lambdadelta", description: "Witch of Certainty" },
];

export function ThemeSelector() {
    const { theme, setTheme, particlesEnabled, setParticlesEnabled } = useTheme();
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
