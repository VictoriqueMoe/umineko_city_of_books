import { useEffect, useRef } from "react";
import type { ThemeType } from "../../../types/app";
import { useTheme } from "../../../hooks/useTheme";
import styles from "./Butterflies.module.css";

const DEFAULT_BUTTERFLY_SYMBOLS = ["\uD83E\uDD8B", "\u2726", "\u2727", "\u271B"];
const DEFAULT_PARTICLE_SYMBOLS = ["\u2726", "\u2727", "\u2B25", "\u25C7"];

const themeSymbols: Partial<Record<ThemeType, { butterflies: string[]; particles: string[] }>> = {
    lambdadelta: {
        butterflies: ["\u2728", "\uD83C\uDF6D", "\uD83C\uDF6C", "\uD83C\uDF83"],
        particles: ["\u2728", "\u2B50", "\uD83C\uDF6D", "\u2726"],
    },
};

const BUTTERFLY_COUNT = 8;
const PARTICLE_COUNT = 15;

function createButterfly(container: HTMLElement, symbols: string[]) {
    const el = document.createElement("div");
    el.className = "butterfly";
    el.textContent = symbols[Math.floor(Math.random() * symbols.length)];
    el.style.setProperty("--start-x", `${Math.random() * 100}vw`);
    el.style.setProperty("--duration", `${15 + Math.random() * 15}s`);
    el.style.setProperty("--delay", `${Math.random() * 5}s`);
    el.style.fontSize = `${0.8 + Math.random() * 1.2}rem`;
    el.addEventListener("animationiteration", () => {
        el.style.setProperty("--start-x", `${Math.random() * 100}vw`);
    });
    container.appendChild(el);
}

function createParticle(container: HTMLElement, symbols: string[]) {
    const el = document.createElement("div");
    el.className = "particle";
    el.textContent = symbols[Math.floor(Math.random() * symbols.length)];
    el.style.setProperty("--start-x", `${Math.random() * 100}vw`);
    el.style.setProperty("--duration", `${15 + Math.random() * 20}s`);
    el.style.setProperty("--delay", `${Math.random() * 10}s`);
    el.style.fontSize = `${0.5 + Math.random() * 0.8}rem`;
    el.addEventListener("animationiteration", () => {
        el.style.setProperty("--start-x", `${Math.random() * 100}vw`);
    });
    container.appendChild(el);
}

export function Butterflies() {
    const containerRef = useRef<HTMLDivElement>(null);
    const { theme } = useTheme();

    useEffect(() => {
        const container = containerRef.current;
        if (!container) {
            return;
        }

        const config = themeSymbols[theme];
        const butterflySymbols = config?.butterflies ?? DEFAULT_BUTTERFLY_SYMBOLS;
        const particleSymbols = config?.particles ?? DEFAULT_PARTICLE_SYMBOLS;

        for (let i = 0; i < BUTTERFLY_COUNT; i++) {
            createButterfly(container, butterflySymbols);
        }
        for (let i = 0; i < PARTICLE_COUNT; i++) {
            createParticle(container, particleSymbols);
        }

        return () => {
            container.innerHTML = "";
        };
    }, [theme]);

    return <div className={styles.container} ref={containerRef} />;
}
