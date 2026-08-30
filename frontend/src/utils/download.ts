export function downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);

    try {
        const link = document.createElement("a");
        link.href = url;
        link.download = filename;

        document.body.appendChild(link);
        link.click();
        link.remove();
    } finally {
        URL.revokeObjectURL(url);
    }
}
