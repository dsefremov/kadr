package media

// ProfileHashV1SDR is the immutable profile set (JPEG q85 3200, WebP 320/640/1280, H.264 CRF23 1080p30).
const ProfileHashV1SDR = "v1sdr20261111"

const (
	VariantMaster      = "master"
	VariantPreview320  = "preview-320"
	VariantPreview640  = "preview-640"
	VariantPreview1280 = "preview-1280"
	VariantVideo       = "video"
	VariantPoster      = "poster"
)
