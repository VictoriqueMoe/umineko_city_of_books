import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { expect, vi } from "vitest";

export function expectInvalidated(queryClient: QueryClient, key: QueryKey): void {
    const invalidateQueries = queryClient.invalidateQueries;

    if (!vi.isMockFunction(invalidateQueries)) {
        throw new Error('expectInvalidated needs vi.spyOn(queryClient, "invalidateQueries") installed before the act');
    }

    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: key });
}
