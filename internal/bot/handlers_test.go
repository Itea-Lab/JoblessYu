package bot

import "testing"

func TestParsePageNumber(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
		valid bool
	}{
		{name: "number", input: "7", want: 7, valid: true},
		{name: "trimmed number", input: " 12 ", want: 12, valid: true},
		{name: "letters", input: "7a", valid: false},
		{name: "signed number", input: "+7", valid: false},
		{name: "empty", input: "", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePageNumber(test.input)
			if (err == nil) != test.valid {
				t.Fatalf("parsePageNumber(%q) error = %v, valid = %v", test.input, err, test.valid)
			}
			if test.valid && got != test.want {
				t.Errorf("parsePageNumber(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}
