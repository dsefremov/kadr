# Media fixtures (CI + stage 4)

| File | License / source | Notes |
|------|------------------|--------|
| `in/sample.heic` | [nokiatech/heif](https://github.com/nokiatech/heif) example assets — **BSD-3-Clause** | Static HEIC for decode/HDR smoke |
| `in/sample.mov` | [apple/media](https://github.com/apple/media) sample HEVC — see repo terms | MOV transcode + tone-map smoke |
| `in/sample.png` | Project-generated test pattern | PNG flatten + resize |

CI: `media-runner smoke`, `scripts/ci-media-pipeline.sh`, Playwright upload flow.
