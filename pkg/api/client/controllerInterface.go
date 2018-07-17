package client

import (
	"net/url"
	"github.com/juju/utils/set"
)

type ControllerInterface interface {
	Put(path string, params url.Values) ([]byte, error)

	Post(path string, op string, params url.Values) ([]byte, error)

	PostFile(path string, op string, params url.Values, fileContent []byte) ([]byte, error)

	Delete(path string) error

	Get(path string, op string, params url.Values) ([]byte, error)

	readAPIVersionInfo() (set.Strings, error)
}
