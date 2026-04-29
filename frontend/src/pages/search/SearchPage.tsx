import { useSearchParams } from "react-router";
import { useSiteSearch } from "../../api/queries/search";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { Button } from "../../components/Button/Button";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { SearchResultRow } from "../../components/layout/GlobalSearch/SearchResultRow";
import { SEARCH_FILTER_OPTIONS } from "../../components/layout/GlobalSearch/searchTypeMeta";
import { usePageTitle } from "../../hooks/usePageTitle";
import styles from "./SearchPage.module.css";

const PAGE_LIMIT = 20;

export function SearchPage() {
    const [params, setParams] = useSearchParams();
    const queryParam = params.get("q") ?? "";
    const typeParam = params.get("type") ?? "";
    const pageParam = Math.max(0, Number(params.get("page") ?? "0"));

    usePageTitle(queryParam ? `Search: ${queryParam}` : "Search");

    const offset = pageParam * PAGE_LIMIT;
    const { results, total, loading, fetching } = useSiteSearch(queryParam, typeParam, PAGE_LIMIT, offset);

    function setQ(next: string) {
        const trimmed = next.trim();
        const newParams = new URLSearchParams(params);
        if (trimmed) {
            newParams.set("q", trimmed);
        } else {
            newParams.delete("q");
        }
        newParams.delete("page");
        setParams(newParams);
    }

    function setType(value: string) {
        const newParams = new URLSearchParams(params);
        if (value) {
            newParams.set("type", value);
        } else {
            newParams.delete("type");
        }
        newParams.delete("page");
        setParams(newParams);
    }

    function setPage(next: number) {
        const newParams = new URLSearchParams(params);
        if (next > 0) {
            newParams.set("page", String(next));
        } else {
            newParams.delete("page");
        }
        setParams(newParams);
    }

    function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
        e.preventDefault();
        const data = new FormData(e.currentTarget);
        setQ(String(data.get("q") ?? ""));
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Search</h1>

            <InfoPanel title="Search Tips">
                <p>
                    Most searches just work — type names, fragments, even typos and misspellings ("beatice" finds
                    "Beatrice"). For when you need more precision:
                </p>
                <ul className={styles.syntaxList}>
                    <li>
                        <code>beatrice battler</code> — both words must appear
                    </li>
                    <li>
                        <code>beatrice OR battler</code> — either word matches
                    </li>
                    <li>
                        <code>witch -hunter</code> — has <strong>witch</strong>, but not <strong>hunter</strong>
                    </li>
                    <li>
                        <code>"golden truth"</code> — exact phrase, words adjacent in order
                    </li>
                    <li>
                        Mix freely: <code>"endless witch" battler -kinzo</code>
                    </li>
                </ul>
                <p>
                    Use the chips below to limit results to a section. <strong>Comments only</strong> spans every
                    section. Drafts, archived journals and banned-user content never appear in results.
                </p>
            </InfoPanel>

            <form className={styles.searchForm} onSubmit={handleSubmit}>
                <Input
                    key={queryParam}
                    name="q"
                    type="search"
                    placeholder="Search anything..."
                    defaultValue={queryParam}
                    fullWidth
                    autoFocus
                />
                <Button type="submit" variant="primary">
                    Search
                </Button>
            </form>

            <div className={styles.filters}>
                {SEARCH_FILTER_OPTIONS.map(opt => (
                    <button
                        key={opt.value || "all"}
                        type="button"
                        className={`${styles.chip} ${typeParam === opt.value ? styles.chipActive : ""}`}
                        onClick={() => setType(opt.value)}
                    >
                        {opt.label}
                    </button>
                ))}
            </div>

            {!queryParam && <div className={styles.hint}>Enter at least 2 characters to search.</div>}

            {queryParam && queryParam.trim().length < 2 && (
                <div className={styles.hint}>Enter at least 2 characters.</div>
            )}

            {queryParam && queryParam.trim().length >= 2 && (
                <>
                    <div className={styles.summary}>
                        {loading
                            ? "Searching..."
                            : `${total} ${total === 1 ? "result" : "results"} for "${queryParam}"`}
                        {fetching && !loading && <span className={styles.refetch}> updating...</span>}
                    </div>

                    {!loading && results.length === 0 && (
                        <div className={styles.empty}>Nothing found. Try different keywords or remove the filter.</div>
                    )}

                    <div className={styles.results}>
                        {results.map(r => (
                            <SearchResultRow key={`${r.type}-${r.id}`} result={r} variant="page" />
                        ))}
                    </div>

                    <Pagination
                        offset={offset}
                        limit={PAGE_LIMIT}
                        total={total}
                        hasNext={offset + PAGE_LIMIT < total}
                        hasPrev={pageParam > 0}
                        onNext={() => setPage(pageParam + 1)}
                        onPrev={() => setPage(Math.max(0, pageParam - 1))}
                    />
                </>
            )}
        </div>
    );
}
