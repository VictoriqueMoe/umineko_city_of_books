import { errorMessage } from "../utils/errorMessage";

export interface UploadFailure {
    label: string;
    reason: string;
}

export const UPLOAD_REFUSED = "the server refused it";

export function uploadFailure(label: string, thrown: unknown): UploadFailure {
    return { label, reason: errorMessage(thrown, UPLOAD_REFUSED) };
}

export function uploadFailureMessage(failures: readonly UploadFailure[]): string {
    if (failures.length === 0) {
        return "";
    }

    if (failures.length === 1) {
        return `${failures[0].label} was not saved: ${failures[0].reason}`;
    }

    const labels = failures.map(failure => failure.label).join(", ");

    return `${failures.length} changes were not saved: ${labels}`;
}
