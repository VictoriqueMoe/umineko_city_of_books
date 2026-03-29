import { useCallback, useEffect, useState } from "react";
import type { TheoryDetail } from "../types/api";
import { getTheory } from "../api/endpoints";

export function useTheory(id: string) {
    const [theory, setTheory] = useState<TheoryDetail | null>(null);
    const [loading, setLoading] = useState(false);

    const fetch = useCallback(async () => {
        setLoading(true);
        try {
            const data = await getTheory(id);
            setTheory(data);
        } catch {
            setTheory(null);
        } finally {
            setLoading(false);
        }
    }, [id]);

    useEffect(() => {
        fetch();
    }, [fetch]);

    return { theory, loading, refresh: fetch };
}
