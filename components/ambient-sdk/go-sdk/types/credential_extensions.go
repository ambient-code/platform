package types

type CredentialTokenResponse struct {
	CredentialID string `json:"credential_id"`
	Provider     string `json:"provider"`
	Token        string `json:"token"`
}
