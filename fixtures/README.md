# Media fixtures (stage 0 smoke)

| File | License / source | Notes |
|------|-------------------|--------|
| `in/sample.heic` | Generated in CI/dev: 64×64 PNG (FFmpeg `lavfi`) → HEIC via Debian `heif-enc` (libheif) | Static HEIC for decode smoke |
| `in/sample.mov` | Generated locally via FFmpeg `lavfi` testsrc + sine | H.264/AAC MOV for transcode smoke |

CI and `media-runner` smoke convert these to JPEG (q85), WebP, and MP4 H.264 to prove the toolchain.
