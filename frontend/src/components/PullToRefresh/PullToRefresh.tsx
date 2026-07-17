import { useEffect, useRef, useState, type ReactNode } from "react";
import { queryClient } from "../../api/queryClient";
import { isNativeApp } from "../../utils/authToken";
import styles from "./PullToRefresh.module.css";

const MAX_PULL = 110;
const THRESHOLD = 70;
const MIN_SPIN_MS = 500;

export function PullToRefresh({ children }: { children: ReactNode }) {
    const [pull, setPull] = useState(0);
    const [refreshing, setRefreshing] = useState(false);
    const [dragging, setDragging] = useState(false);

    const pullRef = useRef(0);
    const refreshingRef = useRef(false);
    const startYRef = useRef<number | null>(null);
    const activeRef = useRef(false);

    useEffect(() => {
        if (!isNativeApp()) {
            return;
        }

        function updatePull(value: number) {
            pullRef.current = value;
            setPull(value);
        }

        function canStart(): boolean {
            return !refreshingRef.current && window.scrollY <= 0 && document.body.dataset.chatPage !== "true";
        }

        function cancel() {
            activeRef.current = false;
            startYRef.current = null;
            setDragging(false);
            updatePull(0);
        }

        function onTouchStart(event: TouchEvent) {
            if (event.touches.length !== 1 || !canStart()) {
                activeRef.current = false;
                return;
            }

            startYRef.current = event.touches[0].clientY;
            activeRef.current = true;
            setDragging(true);
        }

        function onTouchMove(event: TouchEvent) {
            if (!activeRef.current || startYRef.current === null) {
                return;
            }

            const delta = event.touches[0].clientY - startYRef.current;
            if (delta <= 0 || window.scrollY > 0) {
                cancel();
                return;
            }

            const damped = Math.min(delta * 0.5, MAX_PULL);
            updatePull(damped);

            if (event.cancelable) {
                event.preventDefault();
            }
        }

        function finishRefresh() {
            refreshingRef.current = false;
            setRefreshing(false);
            updatePull(0);
        }

        function triggerRefresh() {
            refreshingRef.current = true;
            setRefreshing(true);
            setDragging(false);
            updatePull(THRESHOLD);

            const settle = new Promise<void>(resolve => window.setTimeout(resolve, MIN_SPIN_MS));

            Promise.all([queryClient.refetchQueries({ type: "active" }), settle])
                .catch(() => {})
                .finally(finishRefresh);
        }

        function onTouchEnd() {
            if (!activeRef.current) {
                return;
            }

            activeRef.current = false;
            startYRef.current = null;
            setDragging(false);

            if (pullRef.current >= THRESHOLD) {
                triggerRefresh();
            } else {
                updatePull(0);
            }
        }

        document.addEventListener("touchstart", onTouchStart, { passive: true });
        document.addEventListener("touchmove", onTouchMove, { passive: false });
        document.addEventListener("touchend", onTouchEnd);
        document.addEventListener("touchcancel", onTouchEnd);

        return () => {
            document.removeEventListener("touchstart", onTouchStart);
            document.removeEventListener("touchmove", onTouchMove);
            document.removeEventListener("touchend", onTouchEnd);
            document.removeEventListener("touchcancel", onTouchEnd);
        };
    }, []);

    const ready = pull >= THRESHOLD;
    const visible = pull > 0 || refreshing;

    return (
        <>
            <div
                className={styles.indicator}
                style={{
                    transform: `translateX(-50%) translateY(${pull}px)`,
                    opacity: visible ? 1 : 0,
                    transition: dragging ? "opacity 0.15s ease" : "transform 0.25s ease, opacity 0.2s ease",
                }}
                aria-hidden={!visible}
            >
                <span
                    className={`${styles.spinner} ${refreshing ? styles.spinning : ""} ${ready ? styles.ready : ""}`}
                    style={refreshing ? undefined : { transform: `rotate(${pull * 3}deg)` }}
                />
            </div>
            {children}
        </>
    );
}
