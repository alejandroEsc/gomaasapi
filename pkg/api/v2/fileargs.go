package maasapiv2

import (
	"io"
	"path"

	"github.com/juju/errors"
)

// AddFileArgs is a argument struct for passing information into AddFile.
// One of Content or (Reader, Length) must be specified.
type AddFileArgs struct {
	Filename string
	Content  []byte
	Reader   io.Reader
	Length   int64
}

// Validate checks to make sure the Filename has no slashes, and that one of
// Content or (Reader, Length) is specified.
func (a *AddFileArgs) Validate() error {
	dir, _ := path.Split(a.Filename)
	if dir != "" {
		return errors.NotValidf("paths in Filename %q", a.Filename)
	}
	if a.Filename == "" {
		return errors.NotValidf("missing Filename")
	}
	if a.Content == nil {
		if a.Reader == nil {
			return errors.NotValidf("missing Content or Reader")
		}
		if a.Length == 0 {
			return errors.NotValidf("missing Length")
		}
	} else {
		if a.Reader != nil {
			return errors.NotValidf("specifying Content and Reader")
		}
		if a.Length != 0 {
			return errors.NotValidf("specifying Length and Content")
		}
	}
	return nil
}
