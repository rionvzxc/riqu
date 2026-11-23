package riqu

import "testing"

func TestResponseStatusCategories(t *testing.T) {
	for _, status := range []int{-1, 0, 99, 100, 199, 200, 299, 300, 399, 400, 499, 500, 599, 600} {
		response := &QResponse{statusCode: status}
		got := []bool{response.IsInformational(), response.IsSuccess(), response.IsRedirect(), response.IsClientError(), response.IsServerError()}
		for category, matches := range got {
			want := status >= (category+1)*100 && status < (category+2)*100
			if matches != want {
				t.Errorf("status %d, category %d: got %v, want %v", status, category, matches, want)
			}
		}
	}
	var response *QResponse
	if response.IsInformational() || response.IsSuccess() || response.IsRedirect() || response.IsClientError() || response.IsServerError() {
		t.Error("nil response must not match any status category")
	}
}
