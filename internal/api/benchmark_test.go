package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yxp23/aegis/internal/providers/mock"
	"github.com/Yxp23/aegis/internal/router"
)

func BenchmarkChatHTTP(b *testing.B) {
	mockProvider := &mock.Provider{}
	r := router.New(mockProvider)

	server := httptest.NewServer(NewHandler(r))
	defer server.Close()

	client := server.Client()

	body := []byte(`{
		"model":"mock/mock-model",
		"messages":[
			{"role":"user","content":"benchmark"}
		]
	}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(
			http.MethodPost,
			server.URL+"/v1/chat/completions",
			bytes.NewReader(body),
		)
		if err != nil {
			b.Fatal(err)
		}

		req.Header.Set(
			"Content-Type",
			"application/json",
		)

		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}

		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			resp.Body.Close()
			b.Fatal(err)
		}

		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf(
				"unexpected status: %d",
				resp.StatusCode,
			)
		}
	}
}
func BenchmarkChatHTTPParallel(b *testing.B) {
	mockProvider := &mock.Provider{}
	r := router.New(mockProvider)

	server := httptest.NewServer(NewHandler(r))
	defer server.Close()

	client := server.Client()

	body := []byte(`{
		"model":"mock/mock-model",
		"messages":[
			{"role":"user","content":"benchmark"}
		]
	}`)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(
				http.MethodPost,
				server.URL+"/v1/chat/completions",
				bytes.NewReader(body),
			)
			if err != nil {
				b.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}

			_, err = io.Copy(io.Discard, resp.Body)
			if err != nil {
				resp.Body.Close()
				b.Fatal(err)
			}

			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Fatalf(
					"unexpected status: %d",
					resp.StatusCode,
				)
			}
		}
	})
}
