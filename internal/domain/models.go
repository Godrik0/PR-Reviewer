package domain

import "time"

type User struct {
	UserID   string `json:"user_id" gorm:"primaryKey"`
	Username string `json:"username" gorm:"not null"`
	TeamName string `json:"team_name" gorm:"index;not null"`
	IsActive bool   `json:"is_active" gorm:"default:true"`
}

type Team struct {
	TeamName string `json:"team_name" gorm:"primaryKey"`
	Members  []User `json:"members" gorm:"foreignKey:TeamName;references:TeamName"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type TeamResponse struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	PullRequestID   string     `json:"pull_request_id" gorm:"primaryKey"`
	PullRequestName string     `json:"pull_request_name" gorm:"not null"`
	AuthorID        string     `json:"author_id" gorm:"not null;index"`
	Status          PRStatus   `json:"status" gorm:"type:varchar(10);default:'OPEN'"`
	CreatedAt       *time.Time `json:"createdAt,omitempty" gorm:"autoCreateTime"`
	MergedAt        *time.Time `json:"mergedAt,omitempty"`
}

type PRReviewer struct {
	PullRequestID string `gorm:"primaryKey"`
	ReviewerID    string `gorm:"primaryKey"`
}

type PullRequestResponse struct {
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            PRStatus   `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	PullRequestID   string   `json:"pull_request_id"`
	PullRequestName string   `json:"pull_request_name"`
	AuthorID        string   `json:"author_id"`
	Status          PRStatus `json:"status"`
}

type CreateTeamRequest struct {
	TeamName string       `json:"team_name" binding:"required"`
	Members  []TeamMember `json:"members" binding:"required"`
}

type SetIsActiveRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id" binding:"required"`
	PullRequestName string `json:"pull_request_name" binding:"required"`
	AuthorID        string `json:"author_id" binding:"required"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
}

type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
	OldUserID     string `json:"old_user_id" binding:"required"`
}

type ReassignResponse struct {
	PR         PullRequestResponse `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}

type UserReviewsResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

type DeactivateTeamUsersRequest struct {
	TeamName string   `json:"team_name" binding:"required"`
	UserIDs  []string `json:"user_ids" binding:"required"`
}

type DeactivateTeamUsersResponse struct {
	DeactivatedUsers []string                `json:"deactivated_users"`
	ReassignedPRs    []PRReassignmentSummary `json:"reassigned_prs"`
}

type PRReassignmentSummary struct {
	PullRequestID string   `json:"pull_request_id"`
	OldReviewers  []string `json:"old_reviewers"`
	NewReviewers  []string `json:"new_reviewers"`
}

type PRReassignment struct {
	PullRequestID string
	OldReviewerID string
	NewReviewerID string
}
