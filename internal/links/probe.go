package links

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/michaelginalick/recipe-app/internal/domain"
	"github.com/michaelginalick/recipe-app/internal/recipes"
)

var ErrProbeTimeout = errors.New("probe timed out")

func (lc *LinkClient) Probe(ctx context.Context, rawURL string) (domain.Recipe, error) {
	doc, err, statusCode := fetchHTMLFromLink(lc.Client, rawURL)
	if statusCode == http.StatusTooManyRequests {
		return domain.Recipe{}, ErrTooManyRequests
	}
	if err != nil {
		if isTimeout(err) {
			return domain.Recipe{}, ErrProbeTimeout
		}
		return domain.Recipe{}, err
	}

	return recipes.ParseRecipe(doc), nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
