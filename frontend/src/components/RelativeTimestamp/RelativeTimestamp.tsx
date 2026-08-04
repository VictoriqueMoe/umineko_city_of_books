import {
    formatActiveLabel,
    formatExactDateTime,
    formatMessageTime,
    formatShortDateTime,
    formatTimeOfDay,
    parseServerDate,
    relativeTime,
    shortRelativeTime,
} from "../../utils/time";

type TimestampVariant = "relative" | "short" | "message" | "active" | "timeOnly" | "dateTime";

interface RelativeTimestampProps {
    value: string | null | undefined;
    variant?: TimestampVariant;
    className?: string;
}

const labellers: Record<TimestampVariant, (dateStr: string | null | undefined) => string> = {
    relative: relativeTime,
    short: shortRelativeTime,
    message: formatMessageTime,
    active: formatActiveLabel,
    timeOnly: formatTimeOfDay,
    dateTime: formatShortDateTime,
};

export function RelativeTimestamp({ value, variant = "relative", className }: RelativeTimestampProps) {
    const label = labellers[variant](value);
    const parsed = parseServerDate(value);

    if (!parsed) {
        return <span className={className}>{label}</span>;
    }

    return (
        <time className={className} dateTime={parsed.toISOString()} title={formatExactDateTime(value)}>
            {label}
        </time>
    );
}
