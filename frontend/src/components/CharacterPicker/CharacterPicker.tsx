import { useEffect, useMemo, useState } from "react";
import type { CharacterListEntry, ShipCharacter } from "../../types/api";
import { listCharacters } from "../../api/endpoints";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import { Select } from "../Select/Select";
import styles from "./CharacterPicker.module.css";

type Series = "umineko" | "higurashi" | "oc";

const characterCache = new Map<string, CharacterListEntry[]>();

async function getCharactersCached(series: string): Promise<CharacterListEntry[]> {
    const cached = characterCache.get(series);
    if (cached) {
        return cached;
    }
    const response = await listCharacters(series);
    const sorted = [...response.characters].sort((a, b) => a.name.localeCompare(b.name));
    characterCache.set(series, sorted);
    return sorted;
}

interface SeriesState {
    series: Series;
    list: CharacterListEntry[];
    loading: boolean;
}

function initialState(): SeriesState {
    const cached = characterCache.get("umineko");
    if (cached) {
        return { series: "umineko", list: cached, loading: false };
    }
    return { series: "umineko", list: [], loading: true };
}

interface CharacterPickerProps {
    onAdd: (character: ShipCharacter) => void;
    existing: ShipCharacter[];
    maxCharacters?: number;
}

export function CharacterPicker({ onAdd, existing, maxCharacters }: CharacterPickerProps) {
    const [state, setState] = useState<SeriesState>(initialState);
    const [selectedCanonId, setSelectedCanonId] = useState("");
    const [ocName, setOcName] = useState("");

    const series = state.series;
    const canonList = state.list;
    const loading = state.loading;
    const atLimit = maxCharacters !== undefined && existing.length >= maxCharacters;

    useEffect(() => {
        if (!state.loading || state.series === "oc") {
            return;
        }
        let cancelled = false;
        getCharactersCached(state.series)
            .then(chars => {
                if (!cancelled) {
                    setState(prev =>
                        prev.loading && prev.series === state.series
                            ? { series: prev.series, list: chars, loading: false }
                            : prev,
                    );
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setState(prev =>
                        prev.loading && prev.series === state.series
                            ? { series: prev.series, list: [], loading: false }
                            : prev,
                    );
                }
            });
        return () => {
            cancelled = true;
        };
    }, [state.series, state.loading]);

    function changeSeries(next: Series) {
        setSelectedCanonId("");
        if (next === "oc") {
            setState({ series: "oc", list: [], loading: false });
            return;
        }
        const cached = characterCache.get(next);
        if (cached) {
            setState({ series: next, list: cached, loading: false });
            return;
        }
        setState({ series: next, list: [], loading: true });
    }

    const existingKeys = useMemo(() => {
        const set = new Set<string>();
        for (const c of existing) {
            set.add(`${c.series}:${c.character_id ?? ""}:${c.character_name.toLowerCase()}`);
        }
        return set;
    }, [existing]);

    function handleAdd() {
        if (atLimit) {
            return;
        }
        if (series === "oc") {
            const name = ocName.trim();
            if (!name) {
                return;
            }
            const key = `oc::${name.toLowerCase()}`;
            if (existingKeys.has(key)) {
                return;
            }
            onAdd({
                series: "oc",
                character_name: name,
                sort_order: existing.length,
            });
            setOcName("");
            return;
        }

        const chosen = canonList.find(c => c.id === selectedCanonId);
        if (!chosen) {
            return;
        }
        const key = `${series}:${chosen.id}:${chosen.name.toLowerCase()}`;
        if (existingKeys.has(key)) {
            return;
        }
        onAdd({
            series,
            character_id: chosen.id,
            character_name: chosen.name,
            sort_order: existing.length,
        });
        setSelectedCanonId("");
    }

    if (atLimit) {
        return (
            <div className={styles.picker}>
                <p className={styles.limitMsg}>Maximum {maxCharacters} characters reached.</p>
            </div>
        );
    }

    return (
        <div className={styles.picker}>
            <div className={styles.pickerTabs}>
                <button
                    type="button"
                    className={`${styles.pickerTab}${series === "umineko" ? ` ${styles.pickerTabActive}` : ""}`}
                    onClick={() => changeSeries("umineko")}
                >
                    Umineko
                </button>
                <button
                    type="button"
                    className={`${styles.pickerTab}${series === "higurashi" ? ` ${styles.pickerTabActive}` : ""}`}
                    onClick={() => changeSeries("higurashi")}
                >
                    Higurashi
                </button>
                <button
                    type="button"
                    className={`${styles.pickerTab}${series === "oc" ? ` ${styles.pickerTabActive}` : ""}`}
                    onClick={() => changeSeries("oc")}
                >
                    OC / Other
                </button>
            </div>

            <div className={styles.pickerBody}>
                {series === "oc" ? (
                    <Input
                        type="text"
                        placeholder="Character name..."
                        value={ocName}
                        onChange={e => setOcName(e.target.value)}
                        onKeyDown={e => {
                            if (e.key === "Enter") {
                                e.preventDefault();
                                handleAdd();
                            }
                        }}
                        fullWidth
                    />
                ) : (
                    <Select
                        value={selectedCanonId}
                        onChange={e => setSelectedCanonId(e.target.value)}
                        disabled={loading}
                    >
                        <option value="">{loading ? "Loading..." : "-- choose a character --"}</option>
                        {canonList.map(c => (
                            <option key={c.id} value={c.id}>
                                {c.name}
                            </option>
                        ))}
                    </Select>
                )}
                <Button
                    variant="primary"
                    size="small"
                    onClick={handleAdd}
                    disabled={series === "oc" ? !ocName.trim() : !selectedCanonId}
                >
                    Add
                </Button>
            </div>
        </div>
    );
}
