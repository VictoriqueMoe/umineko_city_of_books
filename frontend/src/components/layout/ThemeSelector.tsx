import { useCallback, useRef, useState } from "react";
import { useTheme } from "../../hooks/useTheme";
import { useClickOutside } from "../../hooks/useClickOutside";
import type { ThemeType } from "../../types/app";
import { ToggleSwitch } from "../common/ToggleSwitch";

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
        <div className="theme-selector" ref={dropdownRef}>
            <button
                className="theme-trigger"
                onClick={() => setIsOpen(!isOpen)}
                aria-expanded={isOpen}
                aria-haspopup="listbox"
            >
                <span className="theme-trigger-label">Theme</span>
                <span className="theme-trigger-sep">{"\u2726"}</span>
                <span className="theme-trigger-name">{currentTheme?.name}</span>
                <span className={`theme-chevron${isOpen ? " open" : ""}`}>{"\u25BC"}</span>
            </button>

            {isOpen && (
                <div className="theme-dropdown" role="listbox">
                    {THEMES.map(t => (
                        <button
                            key={t.id}
                            className={`theme-option${t.id === theme ? " active" : ""}`}
                            onClick={() => {
                                setTheme(t.id);
                                setIsOpen(false);
                            }}
                            role="option"
                            aria-selected={t.id === theme}
                        >
                            <div className="theme-option-info">
                                <span className="theme-option-name">{t.name}</span>
                                <span className="theme-option-desc">{t.description}</span>
                            </div>
                            {t.id === theme && <span className="theme-check">{"\u2713"}</span>}
                        </button>
                    ))}
                    <div className="theme-dropdown-divider" />
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
