package riqu

// TResponseMode controls which redirect responses are delivered to callbacks.
type TResponseMode uint

const (
	// VLastResponse delivers only the final response to the response callback.
	VLastResponse TResponseMode = iota
	// VAllResponses delivers every response in the redirect chain to a response callback.
	VAllResponses
)
