package clients

import "testing"

func TestWithUserDefaultsKeepsWhatTheCredentialsSet(t *testing.T) {
	if got := withUserDefaults(map[string]any{})["insecure"]; got != false {
		t.Errorf("insecure defaults to %v, want false", got)
	}
	if got := withUserDefaults(map[string]any{"insecure": true})["insecure"]; got != true {
		t.Errorf("insecure set by the credentials became %v, want true", got)
	}
}
