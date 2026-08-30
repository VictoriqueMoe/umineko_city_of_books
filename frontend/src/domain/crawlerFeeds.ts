export interface CrawlerFeed {
    name: string;
    url: string;
}

export interface KnownCrawlerFeed extends CrawlerFeed {
    label: string;
}

export const KNOWN_CRAWLER_FEEDS: KnownCrawlerFeed[] = [
    { name: "bing", label: "Bing", url: "https://www.bing.com/toolbox/bingbot.json" },
    {
        name: "google",
        label: "Google",
        url: "https://developers.google.com/static/search/apis/ipranges/googlebot.json",
    },
    { name: "apple", label: "Apple", url: "https://search.developer.apple.com/applebot.json" },
];

const KNOWN_NAMES = new Set(KNOWN_CRAWLER_FEEDS.map(known => known.name));

export function isValidFeedName(name: string): boolean {
    const trimmed = name.trim();

    return trimmed !== "" && !trimmed.includes("=");
}

export function isValidFeedURL(url: string): boolean {
    return url.trim().startsWith("https://");
}

export function parseCrawlerFeeds(raw: string): CrawlerFeed[] {
    const feeds: CrawlerFeed[] = [];

    for (const line of raw.split("\n")) {
        const entry = line.trim();
        const split = entry.indexOf("=");
        if (split <= 0) {
            continue;
        }

        const name = entry.slice(0, split).trim();
        const url = entry.slice(split + 1).trim();
        if (!isValidFeedName(name) || !isValidFeedURL(url)) {
            continue;
        }

        feeds.push({ name, url });
    }

    return feeds;
}

export function serialiseCrawlerFeeds(feeds: CrawlerFeed[]): string {
    return feeds
        .filter(feed => isValidFeedName(feed.name) && isValidFeedURL(feed.url))
        .map(feed => `${feed.name.trim()}=${feed.url.trim()}`)
        .join("\n");
}

export function isKnownFeedEnabled(raw: string, known: KnownCrawlerFeed): boolean {
    return parseCrawlerFeeds(raw).some(feed => feed.name === known.name);
}

export function toggleKnownFeed(raw: string, known: KnownCrawlerFeed, enabled: boolean): string {
    const feeds = parseCrawlerFeeds(raw).filter(feed => feed.name !== known.name);

    if (enabled) {
        feeds.push({ name: known.name, url: known.url });
    }

    return serialiseCrawlerFeeds(feeds);
}

export function customFeeds(raw: string): CrawlerFeed[] {
    return parseCrawlerFeeds(raw).filter(feed => !KNOWN_NAMES.has(feed.name));
}

export function replaceCustomFeeds(raw: string, custom: CrawlerFeed[]): string {
    const kept = parseCrawlerFeeds(raw).filter(feed => KNOWN_NAMES.has(feed.name));

    return serialiseCrawlerFeeds([...kept, ...custom.filter(feed => !KNOWN_NAMES.has(feed.name.trim()))]);
}
