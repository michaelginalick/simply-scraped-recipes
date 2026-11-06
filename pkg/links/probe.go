package links

import (
	"context"
	"errors"
	"net"
	"net/http"

	r "github.com/michaelginalick/recipe-app/pkg/recipes"
)

var ErrProbeTimeout = errors.New("probe timed out")

func (lc *LinkClient) Probe(ctx context.Context, rawURL string) (r.Recipe, error) {
	doc, err, statusCode := fetchHTMLFromLink(lc.Client, rawURL)
	if statusCode == http.StatusTooManyRequests {
		return r.Recipe{}, ErrTooManyRequests
	}
	if err != nil {
		if isTimeout(err) {
			return r.Recipe{}, ErrProbeTimeout
		}
		return r.Recipe{}, err
	}

	return r.ParseRecipe(doc), nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
