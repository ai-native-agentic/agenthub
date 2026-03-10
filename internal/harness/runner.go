package harness

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// GateResult represents a single gate execution result
type GateResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // pass/fail/skip
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output"`
}

// GateResults represents the full gate execution results
type GateResults struct {
	Gates           []GateResult `json:"gates"`
	Overall         string       `json:"overall"` // pass/fail
	TotalDurationMs int64        `json:"total_duration_ms"`
}

// Runner executes Harness gates for a commit
type Runner struct {
	scriptPath string
	db         *sql.DB
}

// NewRunner creates a new Harness runner
func NewRunner(scriptPath string, db *sql.DB) *Runner {
	return &Runner{
		scriptPath: scriptPath,
		db:         db,
	}
}

// RunGatesForCommit executes Harness gates for a commit and stores results
func (r *Runner) RunGatesForCommit(commitHash string, workspaceDir string) (*GateResults, error) {
	startTime := time.Now()

	cmd := exec.Command(
		r.scriptPath,
		"--project-root", workspaceDir,
		"--output-format", "json",
		"--fail-fast",
	)
	cmd.Dir = workspaceDir

	output, _ := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Parse results even if command failed (gates may have failed)
	var results GateResults
	if parseErr := json.Unmarshal(output, &results); parseErr != nil {
		return nil, fmt.Errorf("failed to parse gate results: %w (output: %s)", parseErr, string(output))
	}

	// Store results in database
	for _, gate := range results.Gates {
		_, dbErr := r.db.Exec(`
			INSERT INTO gate_results (commit_hash, gate_name, status, output, duration_ms, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, commitHash, gate.Name, gate.Status, gate.Output, gate.DurationMs, time.Now())

		if dbErr != nil {
			return nil, fmt.Errorf("failed to store gate result: %w", dbErr)
		}
	}

	// Log overall result
	if results.Overall == "pass" {
		log.Printf("Harness gates passed for commit %s (duration: %v)", commitHash, duration)
	} else {
		log.Printf("Harness gates failed for commit %s (duration: %v)", commitHash, duration)
	}

	// Return results even if gates failed (not an error condition)
	return &results, nil
}

// GetGateResults retrieves stored gate results for a commit
func (r *Runner) GetGateResults(commitHash string) (*GateResults, error) {
	rows, err := r.db.Query(`
		SELECT gate_name, status, output, duration_ms
		FROM gate_results
		WHERE commit_hash = ?
		ORDER BY created_at ASC
	`, commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to query gate results: %w", err)
	}
	defer rows.Close()

	var gates []GateResult
	var totalDuration int64

	for rows.Next() {
		var gate GateResult
		if err := rows.Scan(&gate.Name, &gate.Status, &gate.Output, &gate.DurationMs); err != nil {
			return nil, fmt.Errorf("failed to scan gate result: %w", err)
		}
		gates = append(gates, gate)
		totalDuration += gate.DurationMs
	}

	if len(gates) == 0 {
		return nil, fmt.Errorf("no gate results found for commit %s", commitHash)
	}

	// Determine overall status
	overall := "pass"
	for _, gate := range gates {
		if gate.Status == "fail" {
			overall = "fail"
			break
		}
	}

	return &GateResults{
		Gates:           gates,
		Overall:         overall,
		TotalDurationMs: totalDuration,
	}, nil
}
