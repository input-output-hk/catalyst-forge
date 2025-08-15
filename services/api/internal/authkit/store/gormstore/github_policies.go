package gormstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
	"gorm.io/gorm"
)

// GithubPolicy is the GORM model for auth_github_policies.
type GithubPolicy struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Repository    string    `gorm:"index;not null"`
	RefsJSON      string    `gorm:"type:text;column:refs"`
	EnvsJSON      string    `gorm:"type:text;column:environments"`
	WorkflowsJSON string    `gorm:"type:text;column:workflows"`
	RolesJSON     string    `gorm:"type:text;column:roles"`
	Enabled       bool      `gorm:"not null;default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (GithubPolicy) TableName() string { return "auth_github_policies" }

// GithubPolicyStore implements store.GithubPolicyStore.
type GithubPolicyStore struct{ db *gorm.DB }

func NewGithubPolicyStore(db *gorm.DB) *GithubPolicyStore { return &GithubPolicyStore{db: db} }

func (s *GithubPolicyStore) Create(ctx context.Context, p *store.GithubPolicy) error {
	rec, err := toGithubPolicyRec(p)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(&rec).Error
}

func (s *GithubPolicyStore) Update(ctx context.Context, p *store.GithubPolicy) error {
	rec, err := toGithubPolicyRec(p)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&GithubPolicy{}).Where("id = ?", p.ID).Updates(rec).Error
}

func (s *GithubPolicyStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&GithubPolicy{}, "id = ?", id).Error
}

func (s *GithubPolicyStore) GetByID(ctx context.Context, id uuid.UUID) (*store.GithubPolicy, error) {
	var rec GithubPolicy
	if err := s.db.WithContext(ctx).First(&rec, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return toGithubPolicyDomain(&rec)
}

func (s *GithubPolicyStore) List(ctx context.Context) ([]*store.GithubPolicy, error) {
	var recs []GithubPolicy
	if err := s.db.WithContext(ctx).Find(&recs).Error; err != nil {
		return nil, err
	}
	out := make([]*store.GithubPolicy, 0, len(recs))
	for i := range recs {
		d, err := toGithubPolicyDomain(&recs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *GithubPolicyStore) LookupByRepository(ctx context.Context, repository string) ([]*store.GithubPolicy, error) {
	var recs []GithubPolicy
	if err := s.db.WithContext(ctx).Where("repository = ?", repository).Find(&recs).Error; err != nil {
		return nil, err
	}
	out := make([]*store.GithubPolicy, 0, len(recs))
	for i := range recs {
		d, err := toGithubPolicyDomain(&recs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func toGithubPolicyRec(p *store.GithubPolicy) (*GithubPolicy, error) {
	refs, _ := json.Marshal(p.Refs)
	envs, _ := json.Marshal(p.Environments)
	wfs, _ := json.Marshal(p.Workflows)
	roles, _ := json.Marshal(p.Roles)
	return &GithubPolicy{
		ID:            p.ID,
		Repository:    p.Repository,
		RefsJSON:      string(refs),
		EnvsJSON:      string(envs),
		WorkflowsJSON: string(wfs),
		RolesJSON:     string(roles),
		Enabled:       p.Enabled,
	}, nil
}

func toGithubPolicyDomain(rec *GithubPolicy) (*store.GithubPolicy, error) {
	p := &store.GithubPolicy{ID: rec.ID, Repository: rec.Repository, Enabled: rec.Enabled}
	_ = json.Unmarshal([]byte(rec.RefsJSON), &p.Refs)
	_ = json.Unmarshal([]byte(rec.EnvsJSON), &p.Environments)
	_ = json.Unmarshal([]byte(rec.WorkflowsJSON), &p.Workflows)
	_ = json.Unmarshal([]byte(rec.RolesJSON), &p.Roles)
	return p, nil
}
