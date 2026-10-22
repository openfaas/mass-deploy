package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/openfaas/faas-provider/types"
	sdk "github.com/openfaas/go-sdk"
)

func TestDeleteFunctionSDK(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		want   error
	}{
		{"deleted", http.StatusAccepted, nil},
		{"already absent", http.StatusNotFound, nil},
		{"unauthorized", http.StatusUnauthorized, sdk.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, sdk.ErrForbidden},
		{"server error", http.StatusInternalServerError, sdk.ErrUnexpectedStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/system/functions" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				var body types.DeleteFunctionRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.FunctionName != "env-1" || body.Namespace != "test" {
					t.Errorf("unexpected delete: %+v", body)
				}
				w.WriteHeader(tc.status)
				w.Write([]byte("not found in backend"))
			}))
			defer server.Close()
			gateway, _ := url.Parse(server.URL)
			err := deleteFunction(0, "env-1", sdk.NewClient(gateway, nil, server.Client()), "test")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestReconcileSDK(t *testing.T) {
	for _, tc := range []struct {
		name           string
		exists, update bool
		method         string
	}{
		{"create", false, false, http.MethodPost},
		{"update", true, true, http.MethodPut},
		{"skip existing", true, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					if !tc.exists {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"name":"env-1"}`))
					return
				}
				writes++
				if r.Method != tc.method || r.URL.Path != "/system/functions" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				var body types.FunctionDeployment
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Service != "env-1" || body.Namespace != "test" || body.Image != "example/image" || body.EnvProcess != "env" || body.EnvVars["KEY"] != "value" {
					t.Errorf("unexpected spec: %+v", body)
				}
				if body.Labels == nil || (*body.Labels)["label"] != "value" || body.Annotations == nil || (*body.Annotations)["annotation"] != "value" {
					t.Errorf("missing metadata: %+v", body)
				}
				if body.Requests == nil || body.Requests.CPU != "100m" || body.Requests.Memory != "64Mi" || body.Limits == nil || body.Limits.CPU != "200m" || body.Limits.Memory != "128Mi" {
					t.Errorf("unexpected resources: %+v", body)
				}
				w.WriteHeader(http.StatusAccepted)
			}))
			defer server.Close()
			gateway, _ := url.Parse(server.URL)
			err := reconcile(0, "env-1", "example/image", "env", sdk.NewClient(gateway, nil, server.Client()), "test", "create", []string{"KEY=value"}, []string{"label=value"}, []string{"annotation=value"}, tc.update, "100m", "200m", "64Mi", "128Mi")
			if err != nil {
				t.Fatal(err)
			}
			want := 1
			if tc.method == "" {
				want = 0
			}
			if writes != want {
				t.Fatalf("got %d writes, want %d", writes, want)
			}
		})
	}
}

func TestInvokeSDK(t *testing.T) {
	for _, async := range []bool{false, true} {
		t.Run(map[bool]string{false: "sync", true: "async"}[async], func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				prefix := "/function/"
				if async {
					prefix = "/async-function/"
				}
				if r.Method != http.MethodPost || r.URL.Path != prefix+"env-1.test" {
					t.Errorf("unexpected invocation: %s %s", r.Method, r.URL)
				}
				w.WriteHeader(http.StatusAccepted)
			}))
			defer server.Close()
			gateway, _ := url.Parse(server.URL)
			if err := invoke(0, 1, 1, "env-1", "test", sdk.NewClient(gateway, nil, server.Client()), async, time.Second); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("no invocation received")
			}
		})
	}
}
