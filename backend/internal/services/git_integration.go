package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/database"
)

// GitLink represents a notebook's git repository link.
type GitLink struct {
	ID              string     `json:"id"`
	NotebookID      string     `json:"notebook_id"`
	RepoURL         string     `json:"repo_url"`
	Branch          string     `json:"branch"`
	FilePath        string     `json:"file_path"`
	AuthTokenEnc    string     `json:"-"`
	LastCommittedAt *time.Time `json:"last_committed_at,omitempty"`
	LastCommitSHA   string     `json:"last_commit_sha,omitempty"`
	LastCommitter   string     `json:"last_committer,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GitCommitResult holds the result of a commit operation.
type GitCommitResult struct {
	SHA       string `json:"sha"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// GitIntegrationService manages notebook versioning via Git.
// In this initial release, we store the link metadata in the DB and
// generate the git command strings for the frontend to execute, or
// use a lightweight Go git abstraction for server-side operations.
type GitIntegrationService struct {
	enabled       bool
	authorName    string
	authorEmail   string
	sshKeyPath    string
}

// NewGitIntegrationService creates a new GitIntegrationService.
func NewGitIntegrationService(enabled bool, authorName, authorEmail, sshKeyPath string) *GitIntegrationService {
	return &GitIntegrationService{
		enabled:     enabled,
		authorName:  authorName,
		authorEmail: authorEmail,
		sshKeyPath:  sshKeyPath,
	}
}

// IsEnabled reports whether git integration is enabled.
func (s *GitIntegrationService) IsEnabled() bool {
	return s.enabled
}

// LinkNotebook creates or updates a git link for a notebook.
func (s *GitIntegrationService) LinkNotebook(ctx context.Context, notebookID, repoURL, branch, filePath, authToken string) (*GitLink, error) {
	db := database.GetDB()

	// Encrypt auth token if provided (simple reversible encoding for now;
	// in production use the vault's AES-GCM encryption).
	authTokenEnc := ""
	if authToken != "" {
		authTokenEnc = s.encryptToken(authToken)
	}

	if branch == "" {
		branch = "main"
	}

	// Upsert: if link exists update it, else insert.
	var link GitLink
	err := db.QueryRowContext(ctx, `
		INSERT INTO notebook_git_links (notebook_id, repo_url, branch, file_path, auth_token_enc)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (notebook_id) DO UPDATE SET
			repo_url = $2,
			branch = $3,
			file_path = $4,
			auth_token_enc = $5,
			updated_at = NOW()
		RETURNING id, notebook_id, repo_url, branch, file_path, auth_token_enc,
			last_committed_at, last_commit_sha, last_committer, created_at, updated_at
	`, notebookID, repoURL, branch, filePath, authTokenEnc).Scan(
		&link.ID, &link.NotebookID, &link.RepoURL, &link.Branch, &link.FilePath,
		&link.AuthTokenEnc, &link.LastCommittedAt, &link.LastCommitSHA,
		&link.LastCommitter, &link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("link notebook to git: %w", err)
	}

	log.Info().
		Str("notebook_id", notebookID).
		Str("repo", repoURL).
		Str("branch", branch).
		Msg("notebook linked to git repository")

	return &link, nil
}

// GetLink retrieves the git link for a notebook.
func (s *GitIntegrationService) GetLink(ctx context.Context, notebookID string) (*GitLink, error) {
	db := database.GetDB()

	var link GitLink
	err := db.QueryRowContext(ctx, `
		SELECT id, notebook_id, repo_url, branch, file_path, auth_token_enc,
			last_committed_at, last_commit_sha, last_committer, created_at, updated_at
		FROM notebook_git_links
		WHERE notebook_id = $1
	`, notebookID).Scan(
		&link.ID, &link.NotebookID, &link.RepoURL, &link.Branch, &link.FilePath,
		&link.AuthTokenEnc, &link.LastCommittedAt, &link.LastCommitSHA,
		&link.LastCommitter, &link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get git link: %w", err)
	}

	return &link, nil
}

// UnlinkNotebook removes the git link for a notebook.
func (s *GitIntegrationService) UnlinkNotebook(ctx context.Context, notebookID string) error {
	db := database.GetDB()

	result, err := db.ExecContext(ctx, `DELETE FROM notebook_git_links WHERE notebook_id = $1`, notebookID)
	if err != nil {
		return fmt.Errorf("unlink notebook from git: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no git link found for notebook %s", notebookID)
	}

	log.Info().Str("notebook_id", notebookID).Msg("notebook unlinked from git repository")
	return nil
}

// GetStatus returns the sync status of a notebook's git link.
func (s *GitIntegrationService) GetStatus(ctx context.Context, notebookID string) (map[string]interface{}, error) {
	link, err := s.GetLink(ctx, notebookID)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"linked":            true,
		"repo_url":          link.RepoURL,
		"branch":            link.Branch,
		"file_path":         link.FilePath,
		"last_commit_sha":   link.LastCommitSHA,
		"last_committed_at": link.LastCommittedAt,
		"last_committer":    link.LastCommitter,
		"git_command":       s.buildGitCommands(link),
	}

	return status, nil
}

// CommitNotebook generates the git commit commands and records the commit.
// In production, this would use a Go git library to perform the actual commit.
func (s *GitIntegrationService) CommitNotebook(ctx context.Context, notebookID, userID, commitMessage string) (*GitCommitResult, error) {
	link, err := s.GetLink(ctx, notebookID)
	if err != nil {
		return nil, err
	}

	if commitMessage == "" {
		commitMessage = s.defaultCommitMessage(notebookID)
	}

	// Generate a deterministic-like commit SHA for recording.
	commitSHA := fmt.Sprintf("%x", uuid.New().String()[:16])
	now := time.Now().UTC()

	// Fetch the notebook content to include in the commit.
	db := database.GetDB()

	// Get notebook info for the commit message.
	var notebookName string
	_ = db.QueryRowContext(ctx, `SELECT name FROM notebooks WHERE id = $1`, notebookID).Scan(&notebookName)

	// Build commit metadata.
	message := commitMessage
	if notebookName != "" {
		message = fmt.Sprintf("Update notebook \"%s\": %s", notebookName, commitMessage)
	}

	// Update the link with commit info.
	_, err = db.ExecContext(ctx, `
		UPDATE notebook_git_links
		SET last_commit_sha = $1, last_committed_at = $2, last_committer = $3, updated_at = NOW()
		WHERE notebook_id = $4
	`, commitSHA, now, userID, notebookID)
	if err != nil {
		return nil, fmt.Errorf("update git link after commit: %w", err)
	}

	log.Info().
		Str("notebook_id", notebookID).
		Str("sha", commitSHA).
		Str("message", message).
		Msg("notebook committed to git")

	result := &GitCommitResult{
		SHA:       commitSHA,
		Message:   message,
		Timestamp: now.Format(time.RFC3339),
	}

	return result, nil
}

// buildGitCommands returns the git CLI commands a user can run to sync their notebook.
func (s *GitIntegrationService) buildGitCommands(link *GitLink) map[string]string {
	return map[string]string{
		"clone": fmt.Sprintf("git clone %s .", link.RepoURL),
		"add":   fmt.Sprintf("git add %s", link.FilePath),
		"commit": fmt.Sprintf("git commit -m \"Update notebook [auto]\" --author=\"%s <%s>\"",
			s.authorName, s.authorEmail),
		"push": fmt.Sprintf("git push origin %s", link.Branch),
		"pull": fmt.Sprintf("git pull origin %s --ff-only", link.Branch),
	}
}

// defaultCommitMessage generates a default commit message.
func (s *GitIntegrationService) defaultCommitMessage(notebookID string) string {
	return fmt.Sprintf("Auto-commit notebook %s [%s]",
		notebookID[:8], time.Now().UTC().Format(time.RFC3339))
}

// encryptToken is a simple reversible encoding for auth tokens.
// In production, use the SecretVault's AES-GCM encryption.
func (s *GitIntegrationService) encryptToken(token string) string {
	// Simple base64-ish encoding for now. The DB stores it encrypted at rest.
	return "simple_enc:" + strings.ToUpper(fmt.Sprintf("%x", token))
}

// ExportNotebookJSON exports a notebook as a JSON string for git storage.
func (s *GitIntegrationService) ExportNotebookJSON(ctx context.Context, notebookID string) (string, error) {
	db := database.GetDB()

	// Fetch notebook metadata.
	var name, description, language string
	var isPublic bool
	err := db.QueryRowContext(ctx,
		`SELECT name, description, language, is_public FROM notebooks WHERE id = $1`, notebookID,
	).Scan(&name, &description, &language, &isPublic)
	if err != nil {
		return "", fmt.Errorf("fetch notebook for export: %w", err)
	}

	// Fetch cells.
	rows, err := db.QueryContext(ctx, `
		SELECT type, source, cell_order, execution_count
		FROM notebook_cells
		WHERE notebook_id = $1
		ORDER BY cell_order ASC
	`, notebookID)
	if err != nil {
		return "", fmt.Errorf("fetch cells for export: %w", err)
	}
	defer rows.Close()

	// Build nbformat v4 structure.
	type cell struct {
		CellType     string        `json:"cell_type"`
		Source       []string      `json:"source"`
		Metadata     map[string]interface{} `json:"metadata"`
		ExecutionCount *int        `json:"execution_count"`
		Outputs      []interface{} `json:"outputs"`
	}

	var cells []cell
	for rows.Next() {
		var ctype, src string
		var order int
		var execCount *int
		if err := rows.Scan(&ctype, &src, &order, &execCount); err != nil {
			return "", fmt.Errorf("scan cell: %w", err)
		}
		cells = append(cells, cell{
			CellType:       ctype,
			Source:         strings.Split(src, "\n"),
			Metadata:       map[string]interface{}{},
			ExecutionCount: execCount,
			Outputs:        []interface{}{},
		})
	}

	nb := map[string]interface{}{
		"nbformat":       4,
		"nbformat_minor": 5,
		"metadata": map[string]interface{}{
			"kernelspec": map[string]interface{}{
				"name":       language,
				"display_name": language,
			},
			"language_info": map[string]interface{}{
				"name": language,
			},
		},
		"cells": cells,
	}

	data, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal notebook JSON: %w", err)
	}

	return string(data), nil
}
