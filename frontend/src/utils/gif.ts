const GIPHY_URL_RE = /^https:\/\/(media[0-9]*|i)\.giphy\.com\/[^\s]+\.(gif|webp|mp4)(\?[^\s]*)?$/i;
const MEDIA_ID_RE = /\/media\/(?:v\d+\.[^/]+\/)?([a-zA-Z0-9]+)\/[^/]+\.(?:gif|webp|mp4)/i;
const I_SUBDOMAIN_ID_RE = /\/\/i\.giphy\.com\/([a-zA-Z0-9]+)\.(?:gif|webp|mp4)/i;

export function extractGif(body: string): string | null {
    const trimmed = body.trim();
    if (GIPHY_URL_RE.test(trimmed)) {
        return trimmed;
    }
    return null;
}

export function extractGiphyId(url: string): string | null {
    const media = url.match(MEDIA_ID_RE);
    if (media) {
        return media[1];
    }
    const iSub = url.match(I_SUBDOMAIN_ID_RE);
    if (iSub) {
        return iSub[1];
    }
    return null;
}
