export interface MenuPoint {
    x: number;
    y: number;
}

export interface MenuSize {
    width: number;
    height: number;
}

const EDGE_MARGIN = 8;

function clampToViewport(start: number, extent: number, available: number): number {
    const furthest = Math.max(EDGE_MARGIN, available - extent - EDGE_MARGIN);

    return Math.round(Math.min(Math.max(start, EDGE_MARGIN), furthest));
}

export function placeMenu(
    origin: MenuPoint,
    size: MenuSize,
    viewport: MenuSize,
    rightToLeft: boolean = false,
): MenuPoint {
    let x = rightToLeft ? origin.x - size.width : origin.x;

    if (rightToLeft && x < EDGE_MARGIN) {
        x = origin.x;
    }
    if (!rightToLeft && x + size.width > viewport.width - EDGE_MARGIN) {
        x = origin.x - size.width;
    }

    let y = origin.y;
    if (y + size.height > viewport.height - EDGE_MARGIN) {
        y = origin.y - size.height;
    }

    return {
        x: clampToViewport(x, size.width, viewport.width),
        y: clampToViewport(y, size.height, viewport.height),
    };
}
