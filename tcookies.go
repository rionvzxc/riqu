package riqu

// TCookies is a collection of HTTP cookies.
type TCookies []*TCookie

// Set replaces the collection with data.
func (this *TCookies) Set(data TCookies) {
	*this = data
}
// Add appends copies of the supplied cookies to the collection.
func (this *TCookies) Add(cookies ...TCookie) {
	for _, cookie := range cookies {
		cookieCopy := cookie
		*this = append(*this, &cookieCopy)
	}
}
// Del removes every cookie with the given name while preserving order.
func (this *TCookies) Del(name string) {
	cookies := *this
	kept := cookies[:0]

	for _, cookie := range cookies {
		if cookie == nil || cookie.Name != name {
			kept = append(kept, cookie)
		}
	}
	for i := len(kept); i < len(cookies); i++ {
		cookies[i] = nil
	}
	*this = kept
}
// Exist reports whether a cookie with the given name exists.
func (this TCookies) Exist(name string) bool {
	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			return true
		}
	}
	return false
}
// Get returns the first cookie with the given name, or nil if none exists.
func (this TCookies) Get(name string) *TCookie {
	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
// GetAll returns all cookies with the given name.
func (this TCookies) GetAll(name string) TCookies {
	var result TCookies

	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			result = append(result, cookie)
		}
	}
	return result
}
// Clone returns independent copies of the cookies and their Unparsed slices.
func (this TCookies) Clone() TCookies {
	if this == nil {
		return nil
	}
	cookies := make(TCookies, len(this))

	for i, cookie := range this {
		if cookie == nil {
			continue
		}
		clone := *cookie
		clone.Unparsed = append([]string(nil), cookie.Unparsed...)
		cookies[i] = &clone
	}
	return cookies
}
