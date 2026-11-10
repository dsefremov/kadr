package media

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DefaultSpool = "/spool"

// JobManifest is written by the worker for media-runner.
type JobManifest struct {
	JobID     string `json:"job_id"`
	Kind      string `json:"kind"`
	InputName string `json:"input_name"`
}

// JobResult is written by media-runner.
type JobResult struct {
	OK          bool             `json:"ok"`
	ErrorCode   string           `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	Width       int              `json:"width,omitempty"`
	Height      int              `json:"height,omitempty"`
	DurationMs  uint64           `json:"duration_ms,omitempty"`
	Variants    []VariantResult  `json:"variants,omitempty"`
	Metrics     *RunnerMetrics   `json:"metrics,omitempty"`
}

type VariantResult struct {
	Variant string `json:"variant"`
	File    string `json:"file"`
	Mime    string `json:"mime"`
	Bytes   uint64 `json:"bytes"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	SHA256  string `json:"sha256_hex"`
}

type RunnerMetrics struct {
	WallSeconds float64 `json:"wall_seconds"`
	PeakRSSKB   int64   `json:"peak_rss_kb,omitempty"`
}

func SpoolRoot() string {
	if v := os.Getenv("KADR_SPOOL_DIR"); v != "" {
		return v
	}
	return DefaultSpool
}

func PrepareJob(spool, jobID, kind, srcPath string) (jobDir string, err error) {
	jobDir = filepath.Join(spool, "in", jobID)
	if err := os.MkdirAll(jobDir, 0o750); err != nil {
		return "", err
	}
	inName := "input" + filepath.Ext(srcPath)
	if err := copyFile(srcPath, filepath.Join(jobDir, inName)); err != nil {
		return "", err
	}
	outDir := filepath.Join(spool, "out", jobID)
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return "", err
	}
	m := JobManifest{JobID: jobID, Kind: kind, InputName: inName}
	body, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(jobDir, "manifest.json"), body, 0o640); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(jobDir, "ready"), []byte("1"), 0o640); err != nil {
		return "", err
	}
	return jobDir, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = out.ReadFrom(in)
	return err
}

func WaitResult(spool, jobID string, timeout time.Duration) (*JobResult, error) {
	deadline := time.Now().Add(timeout)
	resultPath := filepath.Join(spool, "out", jobID, "result.json")
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(resultPath); err == nil {
			var r JobResult
			if err := json.Unmarshal(data, &r); err != nil {
				return nil, err
			}
			return &r, nil
		}
		failPath := filepath.Join(spool, "out", jobID, "failed")
		if _, err := os.Stat(failPath); err == nil {
			data, _ := os.ReadFile(resultPath)
			var r JobResult
			_ = json.Unmarshal(data, &r)
			if r.ErrorMessage == "" {
				r.ErrorMessage = "processing failed"
			}
			return &r, fmt.Errorf("%w: %s", ErrProcessing, r.ErrorMessage)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("spool timeout for job %s", jobID)
}

func CleanupJob(spool, jobID string) {
	_ = os.RemoveAll(filepath.Join(spool, "in", jobID))
	_ = os.RemoveAll(filepath.Join(spool, "out", jobID))
}
