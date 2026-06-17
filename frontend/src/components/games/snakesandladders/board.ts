export const GRID = 10;
export const CELL = 100;
export const VIEW = GRID * CELL;

export const LADDERS: Record<number, number> = {
    1: 38,
    4: 14,
    9: 31,
    21: 42,
    28: 84,
    36: 44,
    51: 67,
    71: 91,
    80: 100,
};

export const SNAKES: Record<number, number> = {
    16: 6,
    47: 26,
    49: 11,
    56: 53,
    62: 19,
    64: 60,
    87: 24,
    93: 73,
    95: 75,
    98: 78,
};

export interface Point {
    x: number;
    y: number;
}

export function cellColumn(n: number): number {
    const idx = n - 1;
    const row = Math.floor(idx / GRID);
    const within = idx % GRID;
    return row % 2 === 0 ? within : GRID - 1 - within;
}

export function cellRowFromBottom(n: number): number {
    return Math.floor((n - 1) / GRID);
}

export function cellCenter(n: number): Point {
    const col = cellColumn(n);
    const row = cellRowFromBottom(n);
    return {
        x: col * CELL + CELL / 2,
        y: (GRID - 1 - row) * CELL + CELL / 2,
    };
}

export function cellTopLeft(n: number): Point {
    const col = cellColumn(n);
    const row = cellRowFromBottom(n);
    return {
        x: col * CELL,
        y: (GRID - 1 - row) * CELL,
    };
}

function unit(from: Point, to: Point): { ux: number; uy: number; len: number } {
    const dx = to.x - from.x;
    const dy = to.y - from.y;
    const len = Math.hypot(dx, dy) || 1;
    return { ux: dx / len, uy: dy / len, len };
}

export interface LadderGeometry {
    rails: Array<[Point, Point]>;
    rungs: Array<[Point, Point]>;
}

export function buildLadder(from: number, to: number): LadderGeometry {
    const a = cellCenter(from);
    const b = cellCenter(to);
    const { ux, uy, len } = unit(a, b);
    const px = -uy;
    const py = ux;
    const off = 17;

    const railA: [Point, Point] = [
        { x: a.x + px * off, y: a.y + py * off },
        { x: b.x + px * off, y: b.y + py * off },
    ];
    const railB: [Point, Point] = [
        { x: a.x - px * off, y: a.y - py * off },
        { x: b.x - px * off, y: b.y - py * off },
    ];

    const rungs: Array<[Point, Point]> = [];
    const spacing = 46;
    const count = Math.max(2, Math.floor(len / spacing));
    for (let i = 1; i < count; i++) {
        const t = i / count;
        const cx = a.x + ux * len * t;
        const cy = a.y + uy * len * t;
        rungs.push([
            { x: cx + px * off, y: cy + py * off },
            { x: cx - px * off, y: cy - py * off },
        ]);
    }

    return { rails: [railA, railB], rungs };
}

function cubic(p0: Point, p1: Point, p2: Point, p3: Point, t: number): Point {
    const mt = 1 - t;
    const a = mt * mt * mt;
    const b = 3 * mt * mt * t;
    const c = 3 * mt * t * t;
    const d = t * t * t;
    return {
        x: a * p0.x + b * p1.x + c * p2.x + d * p3.x,
        y: a * p0.y + b * p1.y + c * p2.y + d * p3.y,
    };
}

function cubicTangent(p0: Point, p1: Point, p2: Point, p3: Point, t: number): { tx: number; ty: number } {
    const mt = 1 - t;
    const x = 3 * mt * mt * (p1.x - p0.x) + 6 * mt * t * (p2.x - p1.x) + 3 * t * t * (p3.x - p2.x);
    const y = 3 * mt * mt * (p1.y - p0.y) + 6 * mt * t * (p2.y - p1.y) + 3 * t * t * (p3.y - p2.y);
    const len = Math.hypot(x, y) || 1;
    return { tx: x / len, ty: y / len };
}

export interface SnakeGeometry {
    body: string;
    belly: string;
    head: Point;
    headRadius: number;
    eyes: [Point, Point];
    pupilOffset: { x: number; y: number };
    tongue: string;
    angle: number;
}

export function buildSnake(from: number, to: number): SnakeGeometry {
    const head = cellCenter(from);
    const tail = cellCenter(to);
    const { ux, uy, len } = unit(head, tail);
    const px = -uy;
    const py = ux;
    const amp = Math.min(len * 0.22, 140);

    const c1: Point = { x: head.x + ux * len * 0.33 + px * amp, y: head.y + uy * len * 0.33 + py * amp };
    const c2: Point = { x: head.x + ux * len * 0.66 - px * amp, y: head.y + uy * len * 0.66 - py * amp };

    const steps = 28;
    const headHalf = 26;
    const tailHalf = 7;
    const left: Point[] = [];
    const right: Point[] = [];
    for (let i = 0; i <= steps; i++) {
        const t = i / steps;
        const point = cubic(head, c1, c2, tail, t);
        const tan = cubicTangent(head, c1, c2, tail, t);
        const nx = -tan.ty;
        const ny = tan.tx;
        const half = headHalf + (tailHalf - headHalf) * t;
        left.push({ x: point.x + nx * half, y: point.y + ny * half });
        right.push({ x: point.x - nx * half, y: point.y - ny * half });
    }

    let body = `M ${left[0].x.toFixed(1)} ${left[0].y.toFixed(1)}`;
    for (let i = 1; i < left.length; i++) {
        body += ` L ${left[i].x.toFixed(1)} ${left[i].y.toFixed(1)}`;
    }
    for (let i = right.length - 1; i >= 0; i--) {
        body += ` L ${right[i].x.toFixed(1)} ${right[i].y.toFixed(1)}`;
    }
    body += " Z";

    let belly = `M ${head.x.toFixed(1)} ${head.y.toFixed(1)}`;
    for (let i = 1; i <= steps; i++) {
        const t = i / steps;
        const point = cubic(head, c1, c2, tail, t);
        belly += ` L ${point.x.toFixed(1)} ${point.y.toFixed(1)}`;
    }

    const tan0 = cubicTangent(head, c1, c2, tail, 0);
    const outX = -tan0.tx;
    const outY = -tan0.ty;
    const nx0 = -tan0.ty;
    const ny0 = tan0.tx;
    const eyeBase: Point = { x: head.x + outX * headHalf * 0.1, y: head.y + outY * headHalf * 0.1 };
    const eyes: [Point, Point] = [
        { x: eyeBase.x + nx0 * headHalf * 0.45, y: eyeBase.y + ny0 * headHalf * 0.45 },
        { x: eyeBase.x - nx0 * headHalf * 0.45, y: eyeBase.y - ny0 * headHalf * 0.45 },
    ];

    const tongueRoot: Point = { x: head.x + outX * headHalf * 0.9, y: head.y + outY * headHalf * 0.9 };
    const tongueTip: Point = { x: tongueRoot.x + outX * 26, y: tongueRoot.y + outY * 26 };
    const forkA: Point = { x: tongueTip.x + nx0 * 9 + outX * 6, y: tongueTip.y + ny0 * 9 + outY * 6 };
    const forkB: Point = { x: tongueTip.x - nx0 * 9 + outX * 6, y: tongueTip.y - ny0 * 9 + outY * 6 };
    const tongue =
        `M ${tongueRoot.x.toFixed(1)} ${tongueRoot.y.toFixed(1)} ` +
        `L ${tongueTip.x.toFixed(1)} ${tongueTip.y.toFixed(1)} ` +
        `M ${tongueTip.x.toFixed(1)} ${tongueTip.y.toFixed(1)} L ${forkA.x.toFixed(1)} ${forkA.y.toFixed(1)} ` +
        `M ${tongueTip.x.toFixed(1)} ${tongueTip.y.toFixed(1)} L ${forkB.x.toFixed(1)} ${forkB.y.toFixed(1)}`;

    return {
        body,
        belly,
        head,
        headRadius: headHalf + 4,
        eyes,
        pupilOffset: { x: outX * 3, y: outY * 3 },
        tongue,
        angle: Math.atan2(outY, outX),
    };
}
