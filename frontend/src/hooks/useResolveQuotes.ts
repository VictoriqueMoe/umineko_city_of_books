import type { EvidenceItem, Quote, Series } from "../types/api";
import { useEvidenceQuotes } from "./queries/quote";

export function useResolveQuotes(
    evidence: EvidenceItem[],
    series: Series = "umineko",
): ReadonlyMap<string, Quote | null> {
    return useEvidenceQuotes(evidence, series).quotes;
}
