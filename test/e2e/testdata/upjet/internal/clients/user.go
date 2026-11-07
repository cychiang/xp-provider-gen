package clients

// withUserDefaults is the author's own mapping from credentials to the
// Terraform provider's configuration. clients.go calls it.
func withUserDefaults(creds map[string]any) map[string]any {
	if _, ok := creds["insecure"]; !ok {
		creds["insecure"] = false
	}
	return creds
}
