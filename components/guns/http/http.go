package phttp

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type ClientGunConfig struct {
	Target string `validate:"endpoint,required"`
	SSL    bool
	Base   BaseGunConfig `config:",squash"`
}

type HTTPGunConfig struct {
	Gun    ClientGunConfig `config:",squash"`
	Client ClientConfig    `config:",squash"`
}

type HTTP2GunConfig struct {
	Gun    ClientGunConfig `config:",squash"`
	Client ClientConfig    `config:",squash"`
}

func NewHTTPGun(conf HTTPGunConfig, answLog *zap.Logger, targetResolved string) *HTTPGunStruct {
	transport := NewTransport(conf.Client.Transport, NewDialer(conf.Client.Dialer).DialContext, conf.Gun.Target)
	client := newClient(transport, conf.Client.Redirect)
	return NewClientGun(client, conf.Gun, answLog, targetResolved)
}

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(conf HTTP2GunConfig, answLog *zap.Logger, targetResolved string) (*HTTPGunStruct, error) {
	if !conf.Gun.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default.")
	}
	transport := NewHTTP2Transport(conf.Client.Transport, NewDialer(conf.Client.Dialer).DialContext, conf.Gun.Target)
	client := newClient(transport, conf.Client.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	client = &panicOnHTTP1Client{client}
	return NewClientGun(client, conf.Gun, answLog, targetResolved), nil
}

func NewClientGun(client Client, conf ClientGunConfig, answLog *zap.Logger, targetResolved string) *HTTPGunStruct {
	scheme := "http"
	if conf.SSL {
		scheme = "https"
	}
	var httpGun HTTPGunStruct
	httpGun = HTTPGunStruct{
		BaseGun: BaseGun{
			Config: conf.Base,
			Do:     httpGun.Do,
			OnClose: func() error {
				client.CloseIdleConnections()
				return nil
			},
			AnswLog: answLog,
		},
		scheme:         scheme,
		hostname:       getHostWithoutPort(conf.Target),
		targetResolved: targetResolved,
		client:         client,
	}
	return &httpGun
}

type HTTPGunStruct struct {
	BaseGun
	scheme         string
	hostname       string
	targetResolved string
	client         Client
}

var _ Gun = (*HTTPGunStruct)(nil)

func (gun *HTTPGunStruct) Do(req *http.Request) (*http.Response, error) {
	if req.Host == "" {
		req.Host = gun.hostname
	}

	req.URL.Host = gun.targetResolved
	req.URL.Scheme = gun.scheme

	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	requestID := generateRequestID()

	if req.Method == http.MethodPost && req.URL.Path == "/v2/antibot/api/requests" {

		var modifiedBody []byte
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		req.Body.Close()

		var requestBody []map[string]interface{}

		if err = json.Unmarshal(body, &requestBody); err != nil {
			return nil, err
		}

		for i := range requestBody {
			requestBody[i]["request_time"] = timestamp
			requestBody[i]["request_id"] = requestID
		}

		modifiedBody, err = json.Marshal(requestBody)
		if err != nil {
			return nil, err
		}

		req.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
		req.ContentLength = int64(len(modifiedBody))

	}

	return gun.client.Do(req)
}

func DefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		Gun:    DefaultClientGunConfig(),
		Client: DefaultClientConfig(),
	}
}

func DefaultHTTP2GunConfig() HTTP2GunConfig {
	conf := HTTP2GunConfig{
		Client: DefaultClientConfig(),
		Gun:    DefaultClientGunConfig(),
	}
	conf.Gun.SSL = true
	return conf
}

func DefaultClientGunConfig() ClientGunConfig {
	return ClientGunConfig{
		SSL:  false,
		Base: DefaultBaseGunConfig(),
	}
}
func generateRequestID() string {
	rand.Seed(time.Now().UnixNano())
	id := make([]byte, 32)
	for i := 0; i < 32; i++ {
		id[i] = byte('0' + rand.Intn(10))
	}
	return string(id)
}
