package infrai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendLoginCodeRequestAndBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/phone/send_code" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer authorization")
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"message":"phone rejected"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SendLoginCode(context.Background(), "+15551234567", "en")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadRequest || apiErr.Message != "phone rejected" {
		t.Fatalf("error = %#v", err)
	}
}
