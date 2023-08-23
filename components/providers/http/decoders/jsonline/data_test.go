package jsonline

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToRequest(t *testing.T) {
	type want struct {
		method string
		url    string
		header http.Header
		tag    string
		body   []byte
	}
	var tests = []struct {
		name       string
		json       []byte
		confHeader http.Header
		want       want
		wantErr    bool
	}{
		{
			name:       "GET request",
			json:       []byte(`{"host": "wallarm.com", "method": "GET", "uri": "/00", "tag": "tag", "headers": {"A": "a", "B": "b"}}`),
			confHeader: http.Header{"Default": []string{"def"}},
			want:       want{"GET", "http://wallarm.com/00", http.Header{"Default": []string{"def"}, "A": []string{"a"}, "B": []string{"b"}}, "tag", nil},
			wantErr:    false,
		},
		{
			name:       "POST request",
			json:       []byte(`{"host": "wallarm.com", "method": "POST", "uri": "/01?sleep=10", "tag": "tag", "headers": {"A": "a", "B": "b"}, "body": "body"}`),
			confHeader: http.Header{"Default": []string{"def"}},
			want:       want{"POST", "http://wallarm.com/01?sleep=10", http.Header{"Default": []string{"def"}, "A": []string{"a"}, "B": []string{"b"}}, "tag", []byte(`body`)},
			wantErr:    false,
		},
		{
			name:       "POST request with json",
			json:       []byte(`{"host": "wallarm.com", "method": "POST", "uri": "/01?sleep=10", "tag": "tag", "headers": {"A": "a", "B": "b"}, "body": "{\"field\":\"value\"}"}`),
			confHeader: http.Header{"Default": []string{"def"}},
			want:       want{"POST", "http://wallarm.com/01?sleep=10", http.Header{"Default": []string{"def"}, "A": []string{"a"}, "B": []string{"b"}}, "tag", []byte(`{"field":"value"}`)},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertVar := assert.New(t)
			method, url, header, tag, body, err := DecodeAmmo(tt.json, tt.confHeader)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			actual := want{method, url, header, tag, body}
			assertVar.NoError(err)
			assertVar.Equal(tt.want, actual)
		})
	}
}
