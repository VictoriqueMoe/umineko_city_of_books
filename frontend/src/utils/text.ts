export function charCount(value: string): number {
    return [...value].length;
}

export function clampChars(value: string, max: number): string {
    const chars = [...value];
    if (chars.length <= max) {
        return value;
    }

    return chars.slice(0, max).join("");
}

export function ellipsise(value: string, max: number): string {
    const clipped = clampChars(value, max);
    if (clipped === value) {
        return value;
    }

    return clipped + "...";
}
