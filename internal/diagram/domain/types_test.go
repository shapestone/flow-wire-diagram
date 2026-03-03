package domain

import "testing"

func TestVerifyErrorError(t *testing.T) {
	tests := []struct {
		e    VerifyError
		want string
	}{
		{
			e:    VerifyError{Line: 5, Message: "expected │ at col 10"},
			want: "line 5: expected │ at col 10",
		},
		{
			e:    VerifyError{Line: 0, Message: "line too short"},
			want: "line 0: line too short",
		},
		{
			e:    VerifyError{Line: 100, Message: "wide character U+1F30D (🌍) in diagram"},
			want: "line 100: wide character U+1F30D (🌍) in diagram",
		},
	}
	for _, tc := range tests {
		got := tc.e.Error()
		if got != tc.want {
			t.Errorf("VerifyError.Error() = %q, want %q", got, tc.want)
		}
	}
}
