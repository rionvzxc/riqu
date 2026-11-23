package riqu

import "encoding/base64"

// BasicAuth replaces the Authorization header with HTTP Basic credentials.
func (this *QRequest) BasicAuth(username string, password string) *QRequest {
	credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return this.SetHeader("Authorization", "Basic "+credentials)
}

// BearerAuth replaces the Authorization header with a bearer token.
func (this *QRequest) BearerAuth(token string) *QRequest {
	return this.SetHeader("Authorization", "Bearer "+token)
}
