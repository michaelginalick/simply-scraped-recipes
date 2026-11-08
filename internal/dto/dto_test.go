package dto

import "testing"

func TestSubmitJobRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "http://example.com/recipe", false},
		{"valid https", "https://example.com/recipe", false},
		{"empty url", "", true},
		{"no scheme", "example.com/recipe", true},
		{"ftp scheme", "ftp://example.com/recipe", true},
		{"no host", "https://", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := (SubmitJobRequest{URL: c.url}).Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
