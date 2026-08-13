package kernel

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

// DefaultMaxBody is 1 MiB. New toolkit handlers should keep this (or smaller).
// Converted handlers may pass a different MaxBytes; the limit must never be
// raised above what the previous hand-rolled code allowed when one existed.
const DefaultMaxBody int64 = 1 << 20

// DecodeOptions control JSON body reading. Zero value is a safe default for
// new handlers: 1 MiB cap, require application/json, ignore unknown fields
// (the dominant existing behaviour).
type DecodeOptions struct {
	// MaxBytes bounds the request body. Zero means DefaultMaxBody.
	MaxBytes int64
	// RequireJSONContentType rejects missing or non-JSON Content-Type (415).
	// Existing handlers typically did not check Content-Type; converted
	// handlers should leave this false to preserve the contract.
	RequireJSONContentType bool
	// DisallowUnknownFields rejects JSON keys not present on the destination
	// struct. Existing handlers almost never did this; leave false unless the
	// endpoint already rejected unknown fields.
	DisallowUnknownFields bool
	// InvalidJSONMessage overrides the 400 body. Empty uses "Invalid JSON body."
	InvalidJSONMessage string
}

func (o DecodeOptions) maxBytes() int64 {
	if o.MaxBytes > 0 {
		return o.MaxBytes
	}
	return DefaultMaxBody
}

func (o DecodeOptions) invalidJSON() error {
	msg := o.InvalidJSONMessage
	if msg == "" {
		msg = "Invalid JSON body."
	}
	return InvalidInput(msg)
}

// DecodeJSON reads a JSON object into dst. The body is bounded with
// http.MaxBytesReader so an oversized payload returns 413 without exhausting
// the connection. Extra data after the first value is ignored, matching
// encoding/json.Decoder used throughout httpserver today.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, opts DecodeOptions) error {
	if r == nil || r.Body == nil {
		return opts.invalidJSON()
	}
	if opts.RequireJSONContentType && !isJSONContentType(r.Header.Get("Content-Type")) {
		return UnsupportedMediaType()
	}
	r.Body = http.MaxBytesReader(w, r.Body, opts.maxBytes())
	dec := json.NewDecoder(r.Body)
	if opts.DisallowUnknownFields {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(dst); err != nil {
		return mapDecodeError(err, opts)
	}
	return nil
}

func mapDecodeError(err error, opts DecodeOptions) error {
	if err == nil {
		return nil
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return PayloadTooLarge()
	}
	// Go wraps MaxBytesReader failures as json.UnmarshalTypeError / SyntaxError
	// whose Unwrap is not always *MaxBytesError; detect the well-known text.
	if isBodyTooLarge(err) {
		return PayloadTooLarge()
	}
	return opts.invalidJSON()
}

func isBodyTooLarge(err error) bool {
	for err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

func isJSONContentType(ct string) bool {
	if ct == "" {
		return false
	}
	media, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return media == "application/json"
}

// DrainBody discards any unread body so the connection can be reused. Used
// after a 413 so we do not keep reading an attacker-supplied stream.
func DrainBody(r *http.Request) {
	if r == nil || r.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, 1))
	_ = r.Body.Close()
}
