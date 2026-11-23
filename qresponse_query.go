package riqu

import (
	"fmt"
	"net/url"
)

// Query returns the first query value for name or defaultValue when absent.
// Query values are read from the URL that produced this response.
func (this *QResponse) Query(name string, defaultValue string) string {
	if this == nil {
		return defaultValue
	}
	this.parseQuery()
	return this.resQuery.GetDefault(name, defaultValue)
}
// QueryValues returns a copy of all query values for name.
// Invalid query parameters are skipped; BindQuery reports parsing errors.
func (this *QResponse) QueryValues(name string) []string {
	if this == nil {
		return nil
	}
	this.parseQuery()
	return this.resQuery.GetAll(name)
}
// AllQueries returns a clone of the query parameters in this response's URL.
// Invalid query parameters are skipped; BindQuery reports parsing errors.
func (this *QResponse) AllQueries() TValues {
	if this == nil {
		return nil
	}
	this.parseQuery()
	return this.resQuery.Clone()
}
// BindQuery binds query values to struct fields using query tags.
// Parsing errors are returned before binding any fields.
func (this *QResponse) BindQuery(target any) error {
	if this == nil {
		return fmt.Errorf("riqu: response is nil")
	}
	this.parseQuery()

	if this.queryErr != nil {
		return this.queryErr
	}
	return bindValues(target, "query", this.resQuery)
}
func (this *QResponse) parseQuery() {
	this.queryOnce.Do(func() {
		parsedURL, err := url.Parse(this.resUrl)

		if err != nil {
			this.queryErr = fmt.Errorf("riqu: parse response URL: %w", err)
			return
		}
		values, err := url.ParseQuery(parsedURL.RawQuery)
		this.resQuery = TValues(values)

		if err != nil {
			this.queryErr = fmt.Errorf("riqu: parse response query: %w", err)
		}
	})
}
