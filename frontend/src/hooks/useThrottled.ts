import { useCallback, useEffect, useRef } from "react";

export function useThrottled<TArgs extends unknown[]>(
    fn: (...args: TArgs) => void,
    delay: number,
): (...args: TArgs) => void {
    const fnRef = useRef(fn);
    const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const pendingArgsRef = useRef<TArgs | null>(null);

    useEffect(() => {
        fnRef.current = fn;
    }, [fn]);

    useEffect(() => {
        return () => {
            if (timerRef.current !== null) {
                clearTimeout(timerRef.current);
            }
        };
    }, []);

    return useCallback(
        (...args: TArgs) => {
            if (timerRef.current !== null) {
                pendingArgsRef.current = args;
                return;
            }
            fnRef.current(...args);
            const tick = () => {
                if (pendingArgsRef.current !== null) {
                    const pendingArgs = pendingArgsRef.current;
                    pendingArgsRef.current = null;
                    fnRef.current(...pendingArgs);
                    timerRef.current = setTimeout(tick, delay);
                    return;
                }
                timerRef.current = null;
            };
            timerRef.current = setTimeout(tick, delay);
        },
        [delay],
    );
}
