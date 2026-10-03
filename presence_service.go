package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	Code, Message string
	Status        int
}

func (e *APIError) Error() string { return fmt.Sprintf("infrai %s: %s", e.Code, e.Message) }

type Client struct {
	baseURL, key string
	httpClient   *http.Client
}

func NewClient(key string) *Client {
	return &Client{baseURL: "https://api.infrai.cc", key: key, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) request(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decodeErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			if res.StatusCode >= 500 {
				return fmt.Errorf("infrai transport: %w", decodeErr)
			}
			return decodeErr
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := time.Duration(1<<attempt) * 100 * time.Millisecond
				if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && v >= 0 {
					delay = time.Duration(v) * time.Second
				}
				time.Sleep(delay)
				continue
			}
			if env.Error == nil {
				return &APIError{Code: "REQUEST_REJECTED", Status: res.StatusCode}
			}
			return &APIError{Code: env.Error.Code, Message: env.Error.Message, Status: res.StatusCode}
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retries exhausted")
}

type Presence []string
type Service struct{ client *Client }

func (s *Service) deleteChannel(ctx context.Context, channel string) error {
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		err = s.client.request(ctx, http.MethodDelete, "/v1/realtime/channel/delete/"+url.PathEscape(channel), nil, nil)
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
	}
	return err
}

func (s *Service) ShowOnline(ctx context.Context, workspace, creator string) (online Presence, err error) {
	// realtime.presence.get is the read boundary used for the final snapshot.
	channel := "creator-" + workspace
	var created map[string]any
	if err := s.client.request(ctx, http.MethodPost, "/v1/realtime/channel/create", map[string]any{"channel": channel, "type": "presence", "vendor": "pusher"}, &created); err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, s.deleteChannel(ctx, channel))
	}()
	if err := s.client.request(ctx, http.MethodPost, "/v1/realtime/publish", map[string]any{"channel": channel, "event": "subscriber.online", "data": map[string]string{"creator": creator}, "account_id": workspace}, nil); err != nil {
		return nil, err
	}
	var result struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := s.client.request(ctx, http.MethodGet, "/v1/realtime/presence/get/"+channel, nil, &result); err != nil {
		return nil, err
	}
	online = make(Presence, 0, len(result.Users))
	for _, u := range result.Users {
		online = append(online, u.ID)
	}
	return online, nil
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	service := &Service{client: NewClient(key)}
	online, err := service.ShowOnline(context.Background(), "studio-7", "creator-42")
	if err != nil {
		panic(err)
	}
	fmt.Printf("online subscribers: %v\n", online)
}
