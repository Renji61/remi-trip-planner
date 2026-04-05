package httpapp

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestHasMultipartFormData(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if requestHasMultipartFormData(r) {
		t.Fatal("urlencoded should not be multipart")
	}
	r2 := httptest.NewRequest("POST", "/", nil)
	r2.Header.Set("Content-Type", "multipart/form-data; boundary=foo")
	if !requestHasMultipartFormData(r2) {
		t.Fatal("multipart header should match")
	}
}

func TestStoreVehicleImageSkipsNonMultipartBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("title=hello"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	path, err := storeVehicleImage(r, "stop_image", 1<<20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Fatalf("want empty path, got %q", path)
	}
}
