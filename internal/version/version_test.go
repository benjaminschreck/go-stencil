package version

import "testing"

func TestDetails(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "development build without metadata",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			want:    "dev",
		},
		{
			name:    "release build with commit and date",
			version: "v0.1.0",
			commit:  "7abe2887e2d65a5c8e18d6905e491ce4f119d7ec",
			date:    "2026-03-24T10:00:00Z",
			want:    "v0.1.0 (7abe288, 2026-03-24T10:00:00Z)",
		},
		{
			name:    "release build without date",
			version: "v0.1.0",
			commit:  "7abe2887e2d65a5c8e18d6905e491ce4f119d7ec",
			date:    "unknown",
			want:    "v0.1.0 (7abe288)",
		},
	}

	previousVersion := Version
	previousCommit := Commit
	previousDate := Date
	defer func() {
		Version = previousVersion
		Commit = previousCommit
		Date = previousDate
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			Date = tt.date

			if got := Details(); got != tt.want {
				t.Fatalf("Details() = %q, want %q", got, tt.want)
			}
		})
	}
}
