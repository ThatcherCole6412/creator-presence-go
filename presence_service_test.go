package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestShowOnlinePublishesCreatorAndReturnsPresence(t *testing.T) {
	cases := []struct {
		name      string
		workspace string
		creator   string
	}{{name: "studio", workspace: "studio-7", creator: "creator-42"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				var data any
				switch r.URL.Path {
				case "/v1/realtime/channel/create", "/v1/realtime/publish":
					data = map[string]any{}
				case "/v1/realtime/channel/delete/creator-studio-7":
					data = map[string]any{}
				case "/v1/realtime/presence/get/creator-studio-7":
					data = map[string]any{"users": []map[string]string{{"id": "subscriber-1"}, {"id": "subscriber-2"}}}
				}
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": data, "error": nil, "metadata": map[string]any{}})
			}))
			defer server.Close()
			c := NewClient("test-key")
			c.baseURL = server.URL
			got, err := (&Service{client: c}).ShowOnline(context.Background(), tc.workspace, tc.creator)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, Presence{"subscriber-1", "subscriber-2"}) {
				t.Fatalf("presence = %#v", got)
			}
			want := []string{"POST /v1/realtime/channel/create", "POST /v1/realtime/publish", "GET /v1/realtime/presence/get/creator-studio-7", "DELETE /v1/realtime/channel/delete/creator-studio-7"}
			if !reflect.DeepEqual(paths, want) {
				t.Fatalf("paths = %#v", paths)
			}
		})
	}
}

func TestShowOnlineRetriesCleanupAfterPublishFailure(t *testing.T) {
	deleteAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{"ok": true, "data": map[string]any{}, "error": nil, "metadata": map[string]any{}}
		switch r.URL.Path {
		case "/v1/realtime/publish":
			response = map[string]any{"ok": false, "error": map[string]any{"code": "VENDOR_NOT_CONFIGURED", "message": "missing vendor"}}
		case "/v1/realtime/channel/delete/creator-studio-7":
			deleteAttempts++
			if deleteAttempts == 1 {
				response = map[string]any{"ok": false, "error": map[string]any{"code": "REALTIME_CHANNEL_NOT_FOUND", "message": "not visible yet"}}
			}
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	c := NewClient("test-key")
	c.baseURL = server.URL
	_, err := (&Service{client: c}).ShowOnline(context.Background(), "studio-7", "creator-42")
	if err == nil {
		t.Fatal("expected publish error")
	}
	if deleteAttempts != 2 {
		t.Fatalf("delete attempts = %d, want 2", deleteAttempts)
	}
}
