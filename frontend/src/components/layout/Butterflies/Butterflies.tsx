import {useEffect, useRef} from "react";
import styles from "./Butterflies.module.css";

const BUTTERFLY_SYMBOLS = ["\uD83E\uDD8B", "\u2726", "\u2727", "\u271B"];
const BUTTERFLY_COUNT = 8;

const PARTICLE_SYMBOLS = ["\u2726", "\u2727", "\u2B25", "\u25C7"];
const PARTICLE_COUNT = 15;

function createButterfly(container: HTMLElement) {
    const el = document.createElement("div");
    el.className = "butterfly";
    el.textContent = BUTTERFLY_SYMBOLS[Math.floor(Math.random() * BUTTERFLY_SYMBOLS.length)];
    el.style.setProperty("--start-x", `${Math.random() * 100}vw`);
    el.style.setProperty("--duration", `${15 + Math.random() * 15}s`);
    el.style.setProperty("--delay", `${Math.random() * 5}s`);
    el.style.fontSize = `${0.8 + Math.random() * 1.2}rem`;
    el.addEventListener("animationiteration", () => {
        el.style.setProperty("--start-x", `${Math.random() * 100}vw`);
    });
    container.appendChild(el);
}

function createParticle(container: HTMLElement) {
    const el = document.createElement("div");
    el.className = "particle";
    el.textContent = PARTICLE_SYMBOLS[Math.floor(Math.random() * PARTICLE_SYMBOLS.length)];
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

    useEffect(() => {
        const container = containerRef.current;
        if (!container) {
            return;
        }

        for (let i = 0; i < BUTTERFLY_COUNT; i++) {
            createButterfly(container);
        }
        for (let i = 0; i < PARTICLE_COUNT; i++) {
            createParticle(container);
        }

        return () => {
            container.innerHTML = "";
        };
    }, []);

    return <div className={styles.container} ref={containerRef} />;
}
