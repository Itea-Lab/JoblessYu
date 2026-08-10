package job

import (
	"testing"
)

func TestNormalizeLocation(t *testing.T) {
	tests := []struct {
		name        string
		rawLoc      string
		description string
		want        string
	}{
		{
			name:        "ITViec generic Vietnam with Hanoi address in description",
			rawLoc:      "Vietnam",
			description: "Tầng 3, Tháp 1 toà nhà Time Tower, số 35 Lê Văn Lương, Thanh Xuân, Ha Noi",
			want:        "Hanoi",
		},
		{
			name:        "Raw location contains HCM",
			rawLoc:      "Ho Chi Minh City, Vietnam",
			description: "Software developer position in District 1",
			want:        "Ho Chi Minh",
		},
		{
			name:        "Raw location contains Hanoi",
			rawLoc:      "Ha Noi, VN",
			description: "Backend Golang Engineer",
			want:        "Hanoi",
		},
		{
			name:        "Raw location contains Da Nang",
			rawLoc:      "Da Nang City",
			description: "Frontend developer role",
			want:        "Da Nang",
		},
		{
			name:        "Generic Vietnam with Saigon in description",
			rawLoc:      "Vietnam",
			description: "Office located in Sai Gon, near District 1",
			want:        "Ho Chi Minh",
		},
		{
			name:        "Generic Vietnam with Da Nang in description",
			rawLoc:      "Vietnam",
			description: "Working at Hai Chau, Da Nang tech hub",
			want:        "Da Nang",
		},
		{
			name:        "Remote location",
			rawLoc:      "Remote / Work from home",
			description: "Full remote Golang role",
			want:        "Remote",
		},
		{
			name:        "Generic Vietnam without specific city",
			rawLoc:      "Vietnam",
			description: "Seeking talented engineers across the country",
			want:        "Vietnam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLocation(tt.rawLoc, tt.description)
			if got != tt.want {
				t.Errorf("NormalizeLocation(%q, %q) = %q; want %q", tt.rawLoc, tt.description, got, tt.want)
			}
		})
	}
}
