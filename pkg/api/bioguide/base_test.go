package bioguide

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func newClient(url string) *BioGuiden {
	return &BioGuiden{Resty: resty.New(), URL: url, Username: "user&1", Password: "p<w>"}
}

func TestSOAPRequestEscapesCredentials(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.Write([]byte("<ok/>"))
	}))
	defer srv.Close()

	if _, err := newClient(srv.URL).SOAPRequest("x.asmx", "<doc/>"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<username>user&amp;1</username>") || !strings.Contains(got, "<password>p&lt;w&gt;</password>") {
		t.Fatalf("credentials not escaped: %s", got)
	}
}

func TestSOAPRequestErrorResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<Envelope><Body><data><ErrorResult1_0><error-nbr>1003</error-nbr><error-message>Access denied</error-message></ErrorResult1_0></data></Body></Envelope>`))
	}))
	defer srv.Close()

	_, err := newClient(srv.URL).SOAPRequest("x.asmx", "<doc/>")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "1003" || apiErr.Message != "Access denied" {
		t.Fatalf("want APIError 1003, got %v", err)
	}
}

func TestSOAPRequestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`<soap:Fault><faultstring>Server was unable to process</faultstring></soap:Fault>`))
	}))
	defer srv.Close()

	_, err := newClient(srv.URL).SOAPRequest("x.asmx", "<doc/>")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 500 || !strings.Contains(apiErr.Message, "unable to process") {
		t.Fatalf("want 500 APIError, got %v", err)
	}
}
