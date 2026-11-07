export function uploadPercent(sent: number, total: number): number {
    if (total <= 0) {
        return 0;
    }

    return Math.min(100, Math.floor((sent / total) * 100));
}

export function uploadStatus(percent: number): string {
    if (percent < 100) {
        return `Uploading ${percent}%`;
    }

    return "Processing...";
}
