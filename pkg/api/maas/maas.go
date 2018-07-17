package maas

import (
	"fmt"
	"net/url"

	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	v2 "github.com/juju/gomaasapi/pkg/api/v2"
	"github.com/juju/version"
)

type MAAS struct {
	controller client.ControllerInterface
	major      int
}

func NewMASS(baseURL string, apiVersion string, apiKey string) (*MAAS, error) {
	if apiVersion == "" {
		return nil, fmt.Errorf("api version must not be empty")
	}
	major, _, err := version.ParseMajorMinor(apiVersion)
	if err != nil {
		return nil, fmt.Errorf("bad version defined: %s, should be of the form: 2.0", apiVersion)
	}

	switch major {
	case 2:
		c, err := v2.NewControllerWithVersion(baseURL, apiVersion, apiKey)
		if err != nil {
			return nil, err
		}
		fmt.Println("returning new controller with version")
		return &MAAS{controller: c, major: major}, nil

	default:
		return nil, util.NewUnsupportedVersionError("version is not supported: %s", apiVersion)
	}

}

func (m *MAAS) Get(path string, op string, params url.Values) ([]byte, error) {
	return m.controller.Get(path, op, params)
}

func (m *MAAS) Post(path string, op string, params url.Values) ([]byte, error) {
	return m.controller.Post(path, op, params)
}

func (m *MAAS) PostFile(path string, op string, params url.Values, fc []byte) ([]byte, error) {
	return m.controller.PostFile(path, op, params, fc)
}

func (m *MAAS) Put(path string, params url.Values) ([]byte, error) {
	return m.controller.Put(path, params)
}

func (m *MAAS) Delete(path string) error {
	return m.controller.Delete(path)
}
