import { useEffect, useRef } from "react";
import styles from "./MinesweeperCanvas.module.css";

interface LightningBolt {
    segments: { x: number; y: number }[];
    life: number;
    maxLife: number;
    width: number;
    hue: number;
}

function getParticleHue(): number {
    const val = getComputedStyle(document.documentElement).getPropertyValue("--particle-hue").trim();
    return val ? parseFloat(val) : 35;
}

function generateBolt(
    startX: number,
    startY: number,
    endX: number,
    endY: number,
    segmentCount: number,
): { x: number; y: number }[] {
    const points: { x: number; y: number }[] = [{ x: startX, y: startY }];
    for (let i = 1; i < segmentCount; i++) {
        const t = i / segmentCount;
        const baseX = startX + (endX - startX) * t;
        const baseY = startY + (endY - startY) * t;
        const jitter = (1 - Math.abs(t - 0.5) * 2) * 80;
        points.push({
            x: baseX + (Math.random() - 0.5) * jitter,
            y: baseY + (Math.random() - 0.5) * jitter,
        });
    }
    points.push({ x: endX, y: endY });
    return points;
}

function randomEdgePoint(w: number, h: number, cx: number, cy: number): { x: number; y: number } {
    const angle = Math.random() * Math.PI * 2;
    const dist = Math.max(w, h) * 0.6;
    return { x: cx + Math.cos(angle) * dist, y: cy + Math.sin(angle) * dist };
}

interface LightningCanvasProps {
    active: boolean;
}

export function MinesweeperLightningCanvas({ active }: LightningCanvasProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const boltsRef = useRef<LightningBolt[]>([]);
    const animRef = useRef<number>(0);
    const runningRef = useRef(false);
    const spawnTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

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

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) {
            return;
        }
        const ctx = canvas.getContext("2d");
        if (!ctx) {
            return;
        }
        const draw = () => {
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            for (let i = boltsRef.current.length - 1; i >= 0; i--) {
                const bolt = boltsRef.current[i];
                bolt.life++;
                if (bolt.life >= bolt.maxLife) {
                    boltsRef.current.splice(i, 1);
                    continue;
                }
                const alpha = 1 - bolt.life / bolt.maxLife;
                ctx.save();
                ctx.globalAlpha = alpha;
                ctx.strokeStyle = `hsl(${bolt.hue}, 80%, 85%)`;
                ctx.shadowColor = `hsla(${bolt.hue}, 90%, 70%, ${alpha * 0.8})`;
                ctx.shadowBlur = 20;
                ctx.lineWidth = bolt.width * alpha;
                ctx.lineCap = "round";
                ctx.lineJoin = "round";
                ctx.beginPath();
                ctx.moveTo(bolt.segments[0].x, bolt.segments[0].y);
                for (let j = 1; j < bolt.segments.length; j++) {
                    ctx.lineTo(bolt.segments[j].x, bolt.segments[j].y);
                }
                ctx.stroke();
                ctx.strokeStyle = `hsl(${bolt.hue}, 40%, 95%)`;
                ctx.shadowBlur = 0;
                ctx.lineWidth = Math.max(1, bolt.width * alpha * 0.3);
                ctx.beginPath();
                ctx.moveTo(bolt.segments[0].x, bolt.segments[0].y);
                for (let j = 1; j < bolt.segments.length; j++) {
                    ctx.lineTo(bolt.segments[j].x, bolt.segments[j].y);
                }
                ctx.stroke();
                ctx.restore();
            }
            if (boltsRef.current.length > 0) {
                animRef.current = requestAnimationFrame(draw);
            } else {
                runningRef.current = false;
            }
        };

        if (active) {
            const cx = canvas.width / 2;
            const cy = canvas.height / 2;
            const hue = getParticleHue();
            const spawnBolt = () => {
                const end = randomEdgePoint(canvas.width, canvas.height, cx, cy);
                boltsRef.current.push({
                    segments: generateBolt(cx, cy, end.x, end.y, 12 + Math.floor(Math.random() * 8)),
                    life: 0,
                    maxLife: 12 + Math.floor(Math.random() * 8),
                    width: 2 + Math.random() * 3,
                    hue: hue + (Math.random() - 0.5) * 30,
                });
                if (Math.random() < 0.5) {
                    const end2 = randomEdgePoint(canvas.width, canvas.height, cx, cy);
                    boltsRef.current.push({
                        segments: generateBolt(cx, cy, end2.x, end2.y, 10 + Math.floor(Math.random() * 6)),
                        life: 0,
                        maxLife: 10 + Math.floor(Math.random() * 6),
                        width: 1.5 + Math.random() * 2,
                        hue: hue + (Math.random() - 0.5) * 30,
                    });
                }
                if (!runningRef.current) {
                    runningRef.current = true;
                    animRef.current = requestAnimationFrame(draw);
                }
            };
            spawnBolt();
            spawnBolt();
            spawnTimerRef.current = setInterval(spawnBolt, 150);
            const stopTimer = setTimeout(() => {
                if (spawnTimerRef.current) {
                    clearInterval(spawnTimerRef.current);
                    spawnTimerRef.current = null;
                }
            }, 1500);
            return () => {
                clearTimeout(stopTimer);
                if (spawnTimerRef.current) {
                    clearInterval(spawnTimerRef.current);
                    spawnTimerRef.current = null;
                }
                cancelAnimationFrame(animRef.current);
                runningRef.current = false;
            };
        }
        return () => {
            cancelAnimationFrame(animRef.current);
            runningRef.current = false;
        };
    }, [active]);

    return <canvas ref={canvasRef} className={styles.lightningCanvas} />;
}
