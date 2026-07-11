# Task 3.3: Git Integration (Notebook Versioning)

## Mục tiêu
Cho phép user link notebook với Git repository (GitHub/GitLab/Bitbucket) để commit/push/pull version notebook, biến notebook thành code được version control.

## Phạm vi
1. DB migration: `notebook_git_links` table (id, notebook_id, repo_url, branch, file_path, last_committed_at, last_commit_sha, last_committer)
2. Config: `GIT_ENABLED`, `GIT_SSH_KEY_PATH`, `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`
3. Git Service: clone repo vào workspace storage, commit/push notebook, pull latest
4. API endpoints:
   - `POST /api/v1/notebooks/:id/git/link` — link notebook to git file
   - `GET /api/v1/notebooks/:id/git/status` — check sync status
   - `POST /api/v1/notebooks/:id/git/commit` — commit & push
   - `POST /api/v1/notebooks/:id/git/pull` — pull latest version
   - `DELETE /api/v1/notebooks/:id/git/link` — unlink git
5. Storage: git repos cloned into per-user workspace folder `users/{username}/git/`

## Implementation
- Sử dụng `gopkg.in/src-d/go-git.v4` hoặc Go git library
- Clone via HTTPS (token auth) hoặc SSH key
- Notebook export format: .ipynb JSON
- Commit message auto-generated: "Update notebook {name} [{timestamp}]"
