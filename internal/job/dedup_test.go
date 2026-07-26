package job

import (
	"testing"
)

func TestComputeDedupHash(t *testing.T) {
	tests := []struct {
		name      string
		company1  string
		title1    string
		jobType1  string
		company2  string
		title2    string
		jobType2  string
		wantMatch bool
	}{
		{
			name:      "Exact match normalized case and hyphen",
			company1:  "VNG Corporation",
			title1:    "Senior Go Engineer",
			jobType1:  "Full-time",
			company2:  "vng corporation",
			title2:    "Senior Go Engineer",
			jobType2:  "fulltime",
			wantMatch: true,
		},
		{
			name:      "Normalized punctuation & case match",
			company1:  "FPT Software, Co.",
			title1:    "Backend Developer (Golang)",
			jobType1:  "Full-time",
			company2:  "fpt software co",
			title2:    "backend developer golang",
			jobType2:  "fulltime",
			wantMatch: true,
		},
		{
			name:      "Different job type (Full-time vs Part-time)",
			company1:  "VNG",
			title1:    "Backend Developer",
			jobType1:  "Full-time",
			company2:  "vng",
			title2:    "backend developer",
			jobType2:  "Part-time",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := ComputeDedupHash(tt.company1, tt.title1, tt.jobType1)
			hash2 := ComputeDedupHash(tt.company2, tt.title2, tt.jobType2)
			if (hash1 == hash2) != tt.wantMatch {
				t.Errorf("ComputeDedupHash() match = %v, wantMatch = %v (hash1: %s, hash2: %s)", hash1 == hash2, tt.wantMatch, hash1, hash2)
			}
		})
	}
}
