package links

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

type LinkClient struct {
	*http.Client
}

// NewLinkClient returns a LinkClient using the given transport.
// If transport is nil, http.DefaultTransport is used.
func NewLinkClient(transport http.RoundTripper) *LinkClient {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &LinkClient{&http.Client{Transport: transport}}
}

func (link *LinkClient) ExtractRecipeLinks(givenURL, scopedHost string) ([]*url.URL, r.Recipe, error) {
	doc, err := fetchHTMLFromLink(link.Client, givenURL)
	if err != nil || doc == nil {
		return nil, r.Recipe{}, fmt.Errorf("could not fetch html from link %w", err)
	}

	baseURL, err := url.Parse(givenURL)
	if err != nil {
		return nil, r.Recipe{}, fmt.Errorf("invalid base URL: %w", err)
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
					slog.Error("parsing a href tag into url")
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
	return urls, r.ParseRecipe(doc), nil
}

func fetchHTMLFromLink(client *http.Client, rawURL string) (*html.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; recipe-scraper/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return html.Parse(resp.Body)
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