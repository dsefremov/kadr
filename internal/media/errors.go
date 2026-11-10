package media

import "errors"

var (
	ErrNotFound        = errors.New("media not found")
	ErrAnimated        = errors.New("animated image not supported")
	ErrUnsupported     = errors.New("unsupported format")
	ErrVideoTooLong    = errors.New("video exceeds 60 minute limit")
	ErrProcessing      = errors.New("processing failed")
	ErrDuplicate       = errors.New("duplicate upload")
	ErrWaitingQuota    = errors.New("waiting for quota")
)
