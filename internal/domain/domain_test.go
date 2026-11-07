package domain

import "testing"

func TestJobStatus_String(t *testing.T) {
	cases := []struct {
		status JobStatus
		want   string
	}{
		{JobStatusPending, "pending"},
		{JobStatusRunning, "running"},
		{JobStatusDone, "done"},
		{JobStatusError, "error"},
	}

	for _, c := range cases {
		if got := c.status.String(); got != c.want {
			t.Errorf("JobStatus(%d).String() = %q, want %q", c.status, got, c.want)
		}
	}
}

func TestJobStatus_String_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown JobStatus, got none")
		}
	}()
	_ = JobStatus(99).String()
}

func TestParseJobStatus(t *testing.T) {
	cases := []struct {
		input   string
		want    JobStatus
		wantErr bool
	}{
		{"pending", JobStatusPending, false},
		{"running", JobStatusRunning, false},
		{"done", JobStatusDone, false},
		{"error", JobStatusError, false},
		{"unknown", 0, true},
		{"", 0, true},
	}

	for _, c := range cases {
		got, err := ParseJobStatus(c.input)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseJobStatus(%q) error = %v, wantErr %v", c.input, err, c.wantErr)
			continue
		}
		if !c.wantErr && got != c.want {
			t.Errorf("ParseJobStatus(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
