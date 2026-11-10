# Media fixtures (CI + stage 4)

All files in `in/` are committed for reproducible CI. Regenerate with `fixtures/generate-fixtures.sh` (requires `ffmpeg`; HEIC step needs Docker `media-runner` + `heif-enc`).

| File | License | Notes |
|------|---------|--------|
| `in/sample.heic` | **CC0-1.0** (project-generated) | `heif-enc` from lavfi JPEG in `media-runner` |
| `in/sample.mov` | **CC0-1.0** (project-generated) | ffmpeg `testsrc2` + `sine`, 2 s SDR — quick smoke |
| `in/benchmark-hdr-hlg.mov` | **CC0-1.0** (project-generated) | 1080p30, **30 s**, `color_trc=arib-std-b67` (HLG) — HDR→SDR derive benchmark + tests |
| `in/photo-24mp.jpg` | **CC0-1.0** (project-generated) | 6000×4000 jpeg — resize-to-3200 memory benchmark |
| `in/sample.png` | **CC0-1.0** (project-generated) | Tiny PNG for tus / Playwright |

**CC0-1.0:** [Creative Commons Zero](https://creativecommons.org/publicdomain/zero/1.0/) — no rights reserved.

CI: `media-runner smoke`, `scripts/ci-media-pipeline.sh`, `scripts/ci-media-benchmarks.sh`, Playwright upload flow.
