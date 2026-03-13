package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Manager manages database operations
type Manager struct {
	db *sql.DB
	q  *Queries
}

// NewManager creates a new database manager
func NewManager() (*Manager, error) {
	db, err := OpenDB()
	if err != nil {
		return nil, err
	}

	return &Manager{
		db: db,
		q:  New(db),
	}, nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// GetQueries returns the queries interface
func (m *Manager) GetQueries() *Queries {
	return m.q
}

// GetDB returns the underlying database connection
func (m *Manager) GetDB() *sql.DB {
	return m.db
}

// Transaction executes a function within a database transaction
func (m *Manager) Transaction(ctx context.Context, fn func(*Queries) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	q := New(tx)
	if err := fn(q); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rollback err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// Helper methods for common operations

// CreateRepositoryWithDefaults creates a repository with default values
func (m *Manager) CreateRepositoryWithDefaults(ctx context.Context, params CreateRepositoryParams) error {
	now := time.Now().Format(time.RFC3339)
	if params.CreatedAt == "" {
		params.CreatedAt = now
	}
	if params.UpdatedAt == "" {
		params.UpdatedAt = now
	}
	if params.IsDefault == 0 {
		params.IsDefault = 0
	}
	if params.Priority == 0 {
		params.Priority = 0
	}

	return m.q.CreateRepository(ctx, params)
}

// UpdateRepositoryWithTimestamp updates a repository and sets updated_at timestamp
func (m *Manager) UpdateRepositoryWithTimestamp(ctx context.Context, params UpdateRepositoryParams) error {
	now := time.Now().Format(time.RFC3339)
	params.UpdatedAt = now
	return m.q.UpdateRepository(ctx, params)
}

// SetRepositoryAsDefaultByID sets a repository as the default one
func (m *Manager) SetRepositoryAsDefaultByID(ctx context.Context, repoID int64) error {
	return m.Transaction(ctx, func(q *Queries) error {
		// First, unset all default repositories
		if err := q.SetRepositoryAsDefault(ctx); err != nil {
			return err
		}
		
		// Then set the specified repository as default
		now := time.Now().Format(time.RFC3339)
		return q.UpdateRepositoryDefault(ctx, UpdateRepositoryDefaultParams{
			UpdatedAt: now,
			ID:        repoID,
		})
	})
}

// UpdateRepositorySync updates repository sync status
func (m *Manager) UpdateRepositorySync(ctx context.Context, repoID int64, status, errorMsg string) error {
	now := time.Now().Format(time.RFC3339)
	return m.q.UpdateRepositorySyncStatus(ctx, UpdateRepositorySyncStatusParams{
		SyncStatus:   sql.NullString{String: status, Valid: status != ""},
		LastSyncedAt: sql.NullString{String: now, Valid: true},
		SyncError:    sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		UpdatedAt:    now,
		ID:           repoID,
	})
}

// CreateSkillWithDefaults creates a skill with default values
func (m *Manager) CreateSkillWithDefaults(ctx context.Context, params CreateSkillParams) error {
	now := time.Now().Format(time.RFC3339)
	if params.IndexedAt == "" {
		params.IndexedAt = now
	}

	return m.q.CreateSkill(ctx, params)
}

// UpsertSkillBySlug inserts or updates a skill based on repository and slug
func (m *Manager) UpsertSkillBySlug(ctx context.Context, repoID int64, slug string, params CreateSkillParams) (int64, error) {
	// Check if skill exists
	existing, err := m.q.GetSkillByRepositoryAndSlug(ctx, GetSkillByRepositoryAndSlugParams{
		RepositoryID: repoID,
		Slug:         slug,
	})

	if err == nil {
		// Skill exists, update it
		now := time.Now().Format(time.RFC3339)
		updateParams := UpdateSkillParams{
			Description:    params.Description,
			Version:        params.Version,
			Author:         params.Author,
			RepoUrl:        params.RepoUrl,
			RepoOwner:      params.RepoOwner,
			RepoName:       params.RepoName,
			Stars:          params.Stars,
			Forks:          params.Forks,
			Downloads:      params.Downloads,
			Language:       params.Language,
			License:        params.License,
			Homepage:       params.Homepage,
			Category:       params.Category,
			SecurityStatus: params.SecurityStatus,
			UpdatedAt:      sql.NullString{String: now, Valid: true},
			ID:             existing.ID,
		}
		if err := m.q.UpdateSkill(ctx, updateParams); err != nil {
			return 0, err
		}
		return existing.ID, nil
	}

	// Skill doesn't exist, create it
	if err := m.CreateSkillWithDefaults(ctx, params); err != nil {
		return 0, err
	}

	// Get the newly created skill
	created, err := m.q.GetSkillByRepositoryAndSlug(ctx, GetSkillByRepositoryAndSlugParams{
		RepositoryID: repoID,
		Slug:         slug,
	})
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

// AddTagsToSkill adds multiple tags to a skill
func (m *Manager) AddTagsToSkill(ctx context.Context, skillID int64, tags []string) error {
	for _, tag := range tags {
		if err := m.q.CreateTag(ctx, CreateTagParams{
			SkillID: skillID,
			Tag:     tag,
		}); err != nil {
			// Ignore duplicate tag errors
			if !isDuplicateTagError(err) {
				return err
			}
		}
	}
	return nil
}

// ReplaceTagsForSkill replaces all tags for a skill with new ones
func (m *Manager) ReplaceTagsForSkill(ctx context.Context, skillID int64, tags []string) error {
	return m.Transaction(ctx, func(q *Queries) error {
		// Delete existing tags
		if err := q.DeleteTagsBySkillID(ctx, skillID); err != nil {
			return err
		}
		
		// Add new tags
		for _, tag := range tags {
			if err := q.CreateTag(ctx, CreateTagParams{
				SkillID: skillID,
				Tag:     tag,
			}); err != nil {
				if !isDuplicateTagError(err) {
					return err
				}
			}
		}
		
		return nil
	})
}

// isDuplicateTagError checks if the error is due to duplicate tags
func isDuplicateTagError(err error) bool {
	return err != nil && err.Error() == "UNIQUE constraint failed: tags.skill_id, tags.tag"
}

// GetSkillWithTags gets a skill with its associated tags
func (m *Manager) GetSkillWithTags(ctx context.Context, skillID int64) (*SkillWithTags, error) {
	skill, err := m.q.GetSkillByID(ctx, skillID)
	if err != nil {
		return nil, err
	}

	tags, err := m.q.GetTagsBySkillID(ctx, skillID)
	if err != nil {
		return nil, err
	}

	return &SkillWithTags{
		Skill: &skill,
		Tags:  tags,
	}, nil
}

// SkillWithTags represents a skill with its tags
type SkillWithTags struct {
	Skill *Skill
	Tags  []Tag
}