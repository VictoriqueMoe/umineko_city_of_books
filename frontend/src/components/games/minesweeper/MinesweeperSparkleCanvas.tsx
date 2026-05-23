import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import styles from "./MinesweeperCanvas.module.css";

interface Sparkle {
    x: number;
    y: number;
    vx: number;
    vy: number;
    life: number;
    maxLife: number;
    size: number;
    hue: number;
    lightness: number;
}

function getParticleHue(): number {
    const val = getComputedStyle(document.documentElement).getPropertyValue("--particle-hue").trim();
    return val ? parseFloat(val) : 35;
}

export interface SparkleHandle {
    burst: (cx: number, cy: number, count: number) => void;
}

export const MinesweeperSparkleCanvas = forwardRef<SparkleHandle>(function MinesweeperSparkleCanvas(_, ref) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const sparklesRef = useRef<Sparkle[]>([]);
    const animRef = useRef<number>(0);
    const runningRef = useRef(false);
    const drawRef = useRef<() => void>(() => {});

    useEffect(() => {
        drawRef.current = () => {
            const canvas = canvasRef.current;
            if (!canvas) {
                runningRef.current = false;
                return;
            }
            const ctx = canvas.getContext("2d");
            if (!ctx) {
                runningRef.current = false;
                return;
            }
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            for (let i = sparklesRef.current.length - 1; i >= 0; i--) {
                const s = sparklesRef.current[i];
                s.x += s.vx;
                s.y += s.vy;
                s.vy += 0.03;
                s.vx *= 0.98;
                s.life++;
                if (s.life >= s.maxLife) {
                    sparklesRef.current.splice(i, 1);
                    continue;
                }
                const progress = s.life / s.maxLife;
                const alpha = progress < 0.2 ? progress / 0.2 : 1 - (progress - 0.2) / 0.8;
                ctx.save();
                ctx.globalAlpha = alpha * 0.9;
                ctx.fillStyle = `hsl(${s.hue}, 80%, ${s.lightness}%)`;
                ctx.shadowColor = `hsla(${s.hue}, 90%, 70%, 0.8)`;
                ctx.shadowBlur = 6;
                ctx.beginPath();
                ctx.arc(s.x, s.y, s.size * (1 - progress * 0.5), 0, Math.PI * 2);
                ctx.fill();
                ctx.restore();
            }
            if (sparklesRef.current.length > 0) {
                animRef.current = requestAnimationFrame(drawRef.current);
            } else {
                runningRef.current = false;
            }
        };
        return () => {
            cancelAnimationFrame(animRef.current);
        };
    }, []);

    useImperativeHandle(
        ref,
        () => ({
            burst(cx: number, cy: number, count: number) {
                for (let i = 0; i < count; i++) {
                    const angle = Math.random() * Math.PI * 2;
                    const speed = 0.5 + Math.random() * 2.5;
                    sparklesRef.current.push({
                        x: cx + (Math.random() - 0.5) * 6,
                        y: cy + (Math.random() - 0.5) * 6,
                        vx: Math.cos(angle) * speed,
                        vy: Math.sin(angle) * speed,
                        life: 0,
                        maxLife: 30 + Math.random() * 30,
                        size: 1.5 + Math.random() * 2.5,
                        hue: getParticleHue() + Math.random() * 25,
                        lightness: 60 + Math.random() * 20,
                    });
                }
                if (!runningRef.current) {
                    runningRef.current = true;
                    animRef.current = requestAnimationFrame(drawRef.current);
                }
            },
        }),
        [],
    );

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) {
            return;
        }
        const parent = canvas.parentElement;
        if (!parent) {
            return;
        }
        const obs = new ResizeObserver(() => {
            canvas.width = parent.offsetWidth;
            canvas.height = parent.offsetHeight;
        });
        obs.observe(parent);
        canvas.width = parent.offsetWidth;
        canvas.height = parent.offsetHeight;
        return () => obs.disconnect();
    }, []);

    return <canvas ref={canvasRef} className={styles.sparkleCanvas} />;
});
