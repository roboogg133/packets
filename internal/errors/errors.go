package errors_packets

import "errors"

var (
	ErrResponseNot200OK     = errors.New("the request is not 200, download failed")
	ErrCantFindManifestTOML = errors.New("can't find manifest.toml when trying to read the packagefile")
	ErrInvalidSignature     = errors.New("the signature is invalid")
	ErrNotInstalled         = errors.New("the package isn't installed")
	ErrAlredyUpToDate       = errors.New("alredy up to date")
)
