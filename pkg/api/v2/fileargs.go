package maasapiv2

import (
	"io"
)

// AddFileArgs is a argument struct for passing information into AddFile.
// One of Content or (Reader, Length) must be specified.
type AddFileArgs struct {
	Filename string
	Content  []byte
	Reader   io.Reader
	Length   int64
}
