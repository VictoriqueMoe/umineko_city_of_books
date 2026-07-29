package media

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

type (
	Embed struct {
		URL       string
		Type      string
		Title     string
		Desc      string
		Image     string
		SiteName  string
		VideoID   string
		SortOrder int
	}
)

var (
	urlRegex = regexp.MustCompile(`https?://[^\s<>"]+`)
	ytRegex  = regexp.MustCompile(`^https?://(?:[a-z0-9-]+\.)*(?:youtube\.com/(?:watch\?v=|embed/|shorts/)|youtu\.be/)([a-zA-Z0-9_-]{11})`)

	errBlockedAddress = errors.New("media: refusing to dial non-public address")

	previewTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Control: blockNonPublicAddress,
		}).DialContext,
		ForceAttemptHTTP2: true,
		MaxIdleConns:      100,
		IdleConnTimeout:   90 * time.Second,
	}
)

func ExtractURLs(body string) []string {
	return urlRegex.FindAllString(body, -1)
}

func ParseEmbed(rawURL string) *Embed {
	if vid := extractYouTubeID(rawURL); vid != "" {
		return &Embed{
			URL:     rawURL,
			Type:    "youtube",
			VideoID: vid,
		}
	}

	og := fetchOGTags(rawURL)
	if og == nil {
		return nil
	}

	return &Embed{
		URL:      rawURL,
		Type:     "link",
		Title:    og["og:title"],
		Desc:     og["og:description"],
		Image:    og["og:image"],
		SiteName: og["og:site_name"],
	}
}

func extractYouTubeID(rawURL string) string {
	matches := ytRegex.FindStringSubmatch(rawURL)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func blockNonPublicAddress(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return errBlockedAddress
	}

	if !isPublicAddr(addrPort.Addr()) {
		return errBlockedAddress
	}

	return nil
}

func isPublicAddr(addr netip.Addr) bool {
	addr = addr.Unmap()

	if !addr.IsValid() {
		return false
	}

	switch {
	case addr.IsLoopback(),
		addr.IsPrivate(),
		addr.IsUnspecified(),
		addr.IsLinkLocalUnicast(),
		addr.IsLinkLocalMulticast(),
		addr.IsMulticast():
		return false
	}

	return true
}

func fetchOGTags(rawURL string) map[string]string {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil
	}

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: previewTransport,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "UminekoCityOfBooks/1.0 (link preview)")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return nil
	}

	limited := io.LimitReader(resp.Body, 256*1024)
	return parseOGFromHTML(limited)
}

func parseOGFromHTML(r io.Reader) map[string]string {
	tokenizer := html.NewTokenizer(r)
	tags := make(map[string]string)

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}

		t := tokenizer.Token()
		if t.Data != "meta" {
			if t.Data == "body" {
				break
			}
			continue
		}

		var property, content string
		for _, attr := range t.Attr {
			switch attr.Key {
			case "property", "name":
				property = attr.Val
			case "content":
				content = attr.Val
			}
		}

		if strings.HasPrefix(property, "og:") && content != "" {
			if _, exists := tags[property]; !exists {
				tags[property] = content
			}
		}
	}

	return tags
}
