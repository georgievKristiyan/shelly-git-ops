package gitops

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository wraps git operations
type Repository struct {
	repo *git.Repository
	path string
}

// OpenRepository opens a git repository
func OpenRepository(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return &Repository{
		repo: repo,
		path: path,
	}, nil
}

// InitRepository initializes a new git repository
func InitRepository(path string) (*Repository, error) {
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	return &Repository{
		repo: repo,
		path: path,
	}, nil
}

// GetCurrentBranch returns the name of the current branch
func (r *Repository) GetCurrentBranch() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return head.Name().Short(), nil
}

// CreateBranch creates a new branch
func (r *Repository) CreateBranch(branchName string) error {
	head, err := r.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(refName, head.Hash())

	if err := r.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

// CheckoutBranch checks out a branch
func (r *Repository) CheckoutBranch(branchName string) error {
	w, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branchName)

	err = w.Checkout(&git.CheckoutOptions{
		Branch: refName,
		Force:  false,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

// BranchExists checks if a branch exists
func (r *Repository) BranchExists(branchName string) bool {
	refName := plumbing.NewBranchReferenceName(branchName)
	_, err := r.repo.Reference(refName, true)
	return err == nil
}

// AddAll adds all changes to the staging area
func (r *Repository) AddAll() error {
	w, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	return nil
}

// Commit creates a commit with the given message
func (r *Repository) Commit(message string) (string, error) {
	w, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Shelly GitOps",
			Email: "shelly-gitops@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	return hash.String(), nil
}

// HasChanges checks if there are uncommitted changes
func (r *Repository) HasChanges() (bool, error) {
	w, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return !status.IsClean(), nil
}

// GetStatus returns the current repository status
func (r *Repository) GetStatus() (git.Status, error) {
	w, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return w.Status()
}

// Merge merges a branch into the current branch
// Note: go-git doesn't support merge operations directly
// Users should use git CLI for merging with conflict resolution
func (r *Repository) Merge(branchName string) error {
	return fmt.Errorf("merge operation not supported via go-git library. Please use 'git merge %s' command directly for proper conflict resolution", branchName)
}

// GetLog retrieves commit history
func (r *Repository) GetLog(maxCount int) ([]*object.Commit, error) {
	iter, err := r.repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	var commits []*object.Commit
	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if maxCount > 0 && count >= maxCount {
			return fmt.Errorf("limit reached")
		}
		commits = append(commits, c)
		count++
		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, err
	}

	return commits, nil
}

// GetDiffFiles returns list of changed files between two branches
func (r *Repository) GetDiffFiles(fromBranch, toBranch string) ([]string, error) {
	// Get references for both branches
	fromRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(fromBranch), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get from branch reference: %w", err)
	}

	toRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(toBranch), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get to branch reference: %w", err)
	}

	// Get commits
	fromCommit, err := r.repo.CommitObject(fromRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit: %w", err)
	}

	toCommit, err := r.repo.CommitObject(toRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get to commit: %w", err)
	}

	// Get trees
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get from tree: %w", err)
	}

	toTree, err := toCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get to tree: %w", err)
	}

	// Get diff
	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Extract file paths
	var changedFiles []string
	for _, change := range changes {
		// Get the file path (use 'To' for additions/modifications, 'From' for deletions)
		if change.To.Name != "" {
			changedFiles = append(changedFiles, change.To.Name)
		} else if change.From.Name != "" {
			changedFiles = append(changedFiles, change.From.Name)
		}
	}

	return changedFiles, nil
}
