package links

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

func (lc *LinkClient) Probe(ctx context.Context, rawURL string) (domain.Recipe, error) {
	body, err, statusCode := fetchBody(lc.client, rawURL)
	if statusCode == http.StatusTooManyRequests {
		return domain.Recipe{}, domain.ErrTooManyRequests
	}

	if err != nil {
		if isTimeout(err) {
			return domain.Recipe{}, domain.ErrProbeTimeout
		}

		return domain.Recipe{}, err
	}

	defer body.Close()

	return lc.parser.Parse(body), nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
