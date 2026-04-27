import { useMemo, useState } from "react";
import type { ShipCharacter } from "../../types/api";
import { useCharacterList } from "../../api/queries/character";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import { Select } from "../Select/Select";
import styles from "./CharacterPicker.module.css";

type Series = "umineko" | "higurashi" | "ciconia" | "oc";

interface CharacterPickerProps {
    onAdd: (character: ShipCharacter) => void;
    existing: ShipCharacter[];
    maxCharacters?: number;
}

export function CharacterPicker({ onAdd, existing, maxCharacters }: CharacterPickerProps) {
    const [series, setSeries] = useState<Series>("umineko");
    const [selectedCanonId, setSelectedCanonId] = useState("");
    const [ocName, setOcName] = useState("");

    const { characters: rawList, loading } = useCharacterList(series, series !== "oc");
    const canonList = useMemo(() => [...rawList].sort((a, b) => a.name.localeCompare(b.name)), [rawList]);
    const atLimit = maxCharacters !== undefined && existing.length >= maxCharacters;

    function changeSeries(next: Series) {
        setSelectedCanonId("");
        setSeries(next);
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
                    className={`${styles.pickerTab}${series === "ciconia" ? ` ${styles.pickerTabActive}` : ""}`}
                    onClick={() => changeSeries("ciconia")}
                >
                    Ciconia
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
                        {canonList.some(c => c.group === "additional") ? (
                            <>
                                <optgroup label="Main cast">
                                    {canonList
                                        .filter(c => c.group !== "additional")
                                        .map(c => (
                                            <option key={c.id} value={c.id}>
                                                {c.name}
                                            </option>
                                        ))}
                                </optgroup>
                                <optgroup label="Additional">
                                    {canonList
                                        .filter(c => c.group === "additional")
                                        .map(c => (
                                            <option key={c.id} value={c.id}>
                                                {c.name}
                                            </option>
                                        ))}
                                </optgroup>
                            </>
                        ) : (
                            canonList.map(c => (
                                <option key={c.id} value={c.id}>
                                    {c.name}
                                </option>
                            ))
                        )}
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
