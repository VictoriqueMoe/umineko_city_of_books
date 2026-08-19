function formatSize(bytes: number): string {
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
        return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function validateFileSize(
    file: File,
    maxImageSize: number,
    maxVideoSize: number,
    maxAudioSize?: number,
): string | null {
    let kind = "image";
    let maxSize = maxImageSize;

    if (file.type.startsWith("video/")) {
        kind = "video";
        maxSize = maxVideoSize;
    } else if (file.type.startsWith("audio/") && maxAudioSize) {
        kind = "audio";
        maxSize = maxAudioSize;
    }

    if (file.size > maxSize) {
        return `${file.name} is too large (${formatSize(file.size)}). Maximum ${kind} size is ${formatSize(maxSize)}.`;
    }

    return null;
}
