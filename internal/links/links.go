package links

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
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

// NewLinkClient returns a LinkClient using the given transport and parser.
// A nil transport falls back to http.DefaultTransport; a nil parser uses WPRMParser.
func NewLinkClient(transport http.RoundTripper, parser RecipeParser) *LinkClient {
	if transport == nil {
		transport = http.DefaultTransport
	}

	if parser == nil {
		parser = r.WPRMParser{}
	}

	return &LinkClient{
		client: &http.Client{Transport: transport},
		parser: parser,
	}
}

func (link *LinkClient) ExtractRecipeLinks(givenURL, scopedHost string) ([]*url.URL, domain.Recipe, error) {
	body, err, statusCode := fetchBody(link.client, givenURL)

	if statusCode == http.StatusTooManyRequests {
		return nil, domain.Recipe{}, domain.ErrTooManyRequests
	}

	if err != nil || body == nil {
		return nil, domain.Recipe{}, fmt.Errorf("could not fetch html from link %w", err)
	}

	defer body.Close()

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, domain.Recipe{}, fmt.Errorf("reading body: %w", err)
	}

	baseURL, err := url.Parse(givenURL)
	if err != nil {
		return nil, domain.Recipe{}, fmt.Errorf("invalid base URL: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
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

				// ignore bad, non-root, and urls with a query param
				if fullURL.Host != "" && fullURL.Host == scopedHost &&
					len(fullURL.RawQuery) == 0 &&
					!strings.Contains(fullURL.Path, "/de/") &&
					!strings.Contains(fullURL.Path, "comments") {
					urls = append(urls, fullURL)
				}
			}
		}
	}

	forEachNode(doc, visitNode, nil)

	recipe := link.parser.Parse(bytes.NewReader(bodyBytes))

	return urls, recipe, nil
}

func fetchBody(client *http.Client, rawURL string) (io.ReadCloser, error, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err, 0
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; recipe-scraper/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err, 0
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode), resp.StatusCode
	}

	return resp.Body, nil, resp.StatusCode
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
