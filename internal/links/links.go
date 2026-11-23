package links

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/michaelginalick/recipe-app/internal/domain"
	r "github.com/michaelginalick/recipe-app/internal/recipes"
	"golang.org/x/net/html"
)

// RecipeParser extracts a Recipe from an HTML page body.
type RecipeParser interface {
	Parse(io.Reader) domain.Recipe
}

type LinkClient struct {
	client *http.Client
	parser RecipeParser
}

func WithParser(parser RecipeParser) Option {
	return func(lc *LinkClient) error {
		lc.parser = parser
		return nil
	}
}

func WithRoundTripper(c http.RoundTripper) Option {
	return func(lc *LinkClient) error {
		lc.client = &http.Client{Transport: c}
		return nil
	}
}

type Option func(*LinkClient) error

// NewLinkClient returns a LinkClient using the given Options.
// A client without transport falls back to http.DefaultTransport; a client without parser uses WPRMParser.
func NewLinkClient(opts ...Option) (*LinkClient, error) {
	defaultClient := &LinkClient{
		client: &http.Client{Transport: http.DefaultTransport},
		parser: r.WPRMParser{},
	}

	for _, opt := range opts {
		if err := opt(defaultClient); err != nil {
			return &LinkClient{}, nil
		}
	}
	return defaultClient, nil
}

func (link *LinkClient) ExtractRecipeLinks(ctx context.Context, givenURL string) ([]*url.URL, domain.Recipe, error) {
	body, statusCode, err := fetchBody(ctx, link.client, givenURL)

	if statusCode == http.StatusTooManyRequests {
		return nil, domain.Recipe{}, domain.ErrTooManyRequests
	}

	if err != nil || body == nil {
		return nil, domain.Recipe{}, fmt.Errorf("could not fetch html from link %w", err)
	}

	baseURL, err := url.Parse(givenURL)
	if err != nil {
		return nil, domain.Recipe{}, fmt.Errorf("invalid base URL: %w", err)
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, domain.Recipe{}, fmt.Errorf("parsing HTML: %w", err)
	}

	var urls []*url.URL
	visitNode := func(n *html.Node) {
		if n != nil && n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}

				u, err := url.Parse(a.Val)
				if err != nil {
					slog.Error("parsing a href tag into url", "href", a.Val, "error", err)
					continue
				}

				fullURL := baseURL.ResolveReference(u)
				if fullURL.Host != "" && hostsMatch(fullURL.Host, baseURL.Host) &&
					len(fullURL.RawQuery) == 0 &&
					!strings.Contains(fullURL.Path, "/de/") &&
					!strings.Contains(fullURL.Path, "comments") {
					urls = append(urls, fullURL)
				}
			}
		}
	}

	forEachNode(doc, visitNode, nil)
	recipe := link.parser.Parse(bytes.NewReader(body))

	return urls, recipe, nil
}

func fetchBody(ctx context.Context, client *http.Client, rawURL string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; recipe-scraper/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("reading body", "error", err, "errorType", fmt.Sprintf("%T", err))
		return nil, 0, fmt.Errorf("reading body: %w", err)
	}

	return bodyBytes, resp.StatusCode, nil
}

// hostsMatch returns true when a and b refer to the same host, ignoring ports
// and the www. prefix so that www.example.com matches example.com.
func hostsMatch(a, b string) bool {
	normalize := func(h string) string {
		if host, _, err := net.SplitHostPort(h); err == nil {
			h = host
		}
		return strings.TrimPrefix(h, "www.")
	}
	return normalize(a) == normalize(b)
}

// forEachNode recursively visits HTML nodes using the pre/post functions.
func forEachNode(n *html.Node, pre, post func(n *html.Node)) {
	if pre != nil {
		pre(n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, pre, post)
	}

	if post != nil {
		post(n)
	}
}
