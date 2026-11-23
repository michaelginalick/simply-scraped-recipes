package links

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/michaelginalick/recipe-app/internal/domain"
)

func (lc *LinkClient) Probe(ctx context.Context, rawURL string) (domain.Recipe, error) {
	body, statusCode, err := fetchBody(ctx, lc.client, rawURL)
	if statusCode == http.StatusTooManyRequests {
		return domain.Recipe{}, domain.ErrTooManyRequests
	}

	if err != nil {
		if isTimeout(err) {
			return domain.Recipe{}, domain.ErrProbeTimeout
		}

		return domain.Recipe{}, err
	}

	return lc.parser.Parse(bytes.NewReader(body)), nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
