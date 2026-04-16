interface ButterflyProps {
    color: string;
    size?: number;
    className?: string;
}

export function Butterfly({ color, size = 16, className }: ButterflyProps) {
    return (
        <svg width={size} height={size} viewBox="0 0 24 24" className={className} aria-hidden="true" focusable="false">
            <g fill={color} stroke={color} strokeLinejoin="round" strokeWidth="0.5">
                <path
                    d="M12 11 C 10 6, 6 3, 3 5 C 1 7, 2 11, 5 12 C 2 13, 1 16, 3 18 C 6 20, 10 17, 12 13 Z"
                    opacity="0.92"
                />
                <path
                    d="M12 11 C 14 6, 18 3, 21 5 C 23 7, 22 11, 19 12 C 22 13, 23 16, 21 18 C 18 20, 14 17, 12 13 Z"
                    opacity="0.92"
                />
            </g>
            <path
                d="M12 5 Q 12 12 12 19"
                stroke={color}
                strokeWidth="1.1"
                strokeLinecap="round"
                fill="none"
                opacity="0.85"
            />
            <circle cx="12" cy="5" r="1" fill={color} />
        </svg>
    );
}
