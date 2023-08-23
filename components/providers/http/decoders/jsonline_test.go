package decoders

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wallarm/specter/components/providers/http/config"
	"github.com/wallarm/specter/components/providers/http/decoders/ammo"
)

const jsonlineDecoderInput = `{"host": "wallarm.com", "method": "GET", "uri": "/?sleep=100", "tag": "sleep1", "headers": {"User-agent": "Tank", "Connection": "close"}}
{"host": "wallarm.com", "method": "POST", "uri": "/?sleep=200", "tag": "sleep2", "headers": {"User-agent": "Tank", "Connection": "close"}, "body": "body_data"}


`

func getJsonlineAmmoWants(t *testing.T) []DecodedAmmo {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}
	return []DecodedAmmo{
		mustNewAmmo(t,
			"GET",
			"http://wallarm.com/?sleep=100",
			nil,
			http.Header{"Connection": []string{"close"}, "Content-Type": []string{"application/json"}, "User-Agent": []string{"Tank"}},
			"sleep1",
		),
		mustNewAmmo(t,
			"POST",
			"http://wallarm.com/?sleep=200",
			[]byte("body_data"),
			http.Header{"Connection": []string{"close"}, "Content-Type": []string{"application/json"}, "User-Agent": []string{"Tank"}},
			"sleep2",
		),
	}
}

func Test_jsonlineDecoder_Scan(t *testing.T) {
	decoder := newJsonlineDecoder(strings.NewReader(jsonlineDecoderInput), config.Config{
		Limit: 4,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wants := getJsonlineAmmoWants(t)
	for j := 0; j < 2; j++ {
		for i, want := range wants {
			ammo, err := decoder.Scan(ctx)
			assert.NoError(t, err, "iteration %d-%d", j, i)
			assert.Equal(t, want, ammo, "iteration %d-%d", j, i)
		}
	}

	_, err := decoder.Scan(ctx)
	assert.Equal(t, err, ErrAmmoLimit)
	assert.Equal(t, decoder.ammoNum, uint(len(wants)*2))
	assert.Equal(t, decoder.passNum, uint(1))
}

func Test_jsonlineDecoder_LoadAmmo(t *testing.T) {
	decoder := newJsonlineDecoder(strings.NewReader(jsonlineDecoderInput), config.Config{
		Limit: 4,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wants := getJsonlineAmmoWants(t)

	ammos, err := decoder.LoadAmmo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, wants, ammos)
	assert.Equal(t, decoder.config.Limit, uint(4))
	assert.Equal(t, decoder.config.Passes, uint(0))
}
