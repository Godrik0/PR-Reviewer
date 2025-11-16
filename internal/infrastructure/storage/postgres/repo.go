package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"pr-reviewer/internal/domain"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(dsn string) (*PostgresRepository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&domain.Team{}, &domain.User{}, &domain.PullRequest{}, &domain.PRReviewer{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	db.Exec("CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer ON pr_reviewers(reviewer_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active)")

	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (r *PostgresRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *PostgresRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := getTx(ctx); tx != nil {
		return tx
	}

	return r.db
}

func (r *PostgresRepository) CreateTeam(ctx context.Context, team *domain.Team, members []domain.User) error {
	db := r.getDB(ctx)

	if err := db.Create(&domain.Team{TeamName: team.TeamName}).Error; err != nil {
		return err
	}

	for i := range members {
		members[i].TeamName = team.TeamName
		if err := db.Save(&members[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresRepository) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	db := r.getDB(ctx)

	var team domain.Team
	if err := db.Preload("Members").Where("team_name = ?", teamName).First(&team).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTeamNotFound
		}
		return nil, err
	}

	return &team, nil
}

func (r *PostgresRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	db := r.getDB(ctx)

	var count int64
	if err := db.Model(&domain.Team{}).Where("team_name = ?", teamName).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostgresRepository) CreateOrUpdateUser(ctx context.Context, user *domain.User) error {
	db := r.getDB(ctx)
	return db.Save(user).Error
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	db := r.getDB(ctx)

	var user domain.User
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) GetUsersByTeam(ctx context.Context, teamName string) ([]domain.User, error) {
	db := r.getDB(ctx)

	var users []domain.User
	if err := db.Where("team_name = ?", teamName).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *PostgresRepository) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	db := r.getDB(ctx)
	result := db.Model(&domain.User{}).Where("user_id = ?", userID).Update("is_active", isActive)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *PostgresRepository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error) {
	db := r.getDB(ctx)

	var users []domain.User
	query := db.Where("team_name = ? AND is_active = ?", teamName, true)

	if excludeUserID != "" {
		query = query.Where("user_id != ?", excludeUserID)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *PostgresRepository) CreatePR(ctx context.Context, pr *domain.PullRequest, reviewers []string) error {
	db := r.getDB(ctx)

	if err := db.Create(pr).Error; err != nil {
		return err
	}

	for _, reviewerID := range reviewers {
		prReviewer := domain.PRReviewer{
			PullRequestID: pr.PullRequestID,
			ReviewerID:    reviewerID,
		}
		if err := db.Create(&prReviewer).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresRepository) GetPR(ctx context.Context, prID string) (*domain.PullRequest, error) {
	db := r.getDB(ctx)

	var pr domain.PullRequest
	if err := db.Where("pull_request_id = ?", prID).First(&pr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrPRNotFound
		}
		return nil, err
	}

	return &pr, nil
}

func (r *PostgresRepository) GetPRWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, []string, error) {
	pr, err := r.GetPR(ctx, prID)
	if err != nil {
		return nil, nil, err
	}

	reviewers, err := r.GetPRReviewers(ctx, prID)
	if err != nil {
		return nil, nil, err
	}

	return pr, reviewers, nil
}

func (r *PostgresRepository) PRExists(ctx context.Context, prID string) (bool, error) {
	db := r.getDB(ctx)

	var count int64
	if err := db.Model(&domain.PullRequest{}).Where("pull_request_id = ?", prID).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostgresRepository) MergePR(ctx context.Context, prID string) error {
	db := r.getDB(ctx)

	now := time.Now()
	result := db.Model(&domain.PullRequest{}).
		Where("pull_request_id = ?", prID).
		Updates(map[string]interface{}{
			"status":    domain.PRStatusMerged,
			"merged_at": now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrPRNotFound
	}

	return nil
}

func (r *PostgresRepository) GetPRReviewers(ctx context.Context, prID string) ([]string, error) {
	db := r.getDB(ctx)

	var prReviewers []domain.PRReviewer
	if err := db.Where("pull_request_id = ?", prID).Find(&prReviewers).Error; err != nil {
		return nil, err
	}

	reviewerIDs := make([]string, len(prReviewers))
	for i, pr := range prReviewers {
		reviewerIDs[i] = pr.ReviewerID
	}

	return reviewerIDs, nil
}

func (r *PostgresRepository) AddReviewer(ctx context.Context, prID, userID string) error {
	db := r.getDB(ctx)
	prReviewer := domain.PRReviewer{
		PullRequestID: prID,
		ReviewerID:    userID,
	}

	return db.Create(&prReviewer).Error
}

func (r *PostgresRepository) RemoveReviewer(ctx context.Context, prID, userID string) error {
	db := r.getDB(ctx)

	return db.Where("pull_request_id = ? AND reviewer_id = ?", prID, userID).
		Delete(&domain.PRReviewer{}).Error
}

func (r *PostgresRepository) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	db := r.getDB(ctx)

	var prs []domain.PullRequest
	err := db.
		Joins("JOIN pr_reviewers ON pr_reviewers.pull_request_id = pull_requests.pull_request_id").
		Where("pr_reviewers.reviewer_id = ?", userID).
		Find(&prs).Error

	if err != nil {
		return nil, err
	}

	return prs, nil
}

func (r *PostgresRepository) IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error) {
	db := r.getDB(ctx)

	var count int64
	if err := db.Model(&domain.PRReviewer{}).
		Where("pull_request_id = ? AND reviewer_id = ?", prID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostgresRepository) DeactivateUsers(ctx context.Context, userIDs []string) error {
	db := r.getDB(ctx)

	result := db.Model(&domain.User{}).
		Where("user_id IN ?", userIDs).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *PostgresRepository) GetOpenPRsWithReviewers(ctx context.Context, reviewerIDs []string) ([]domain.PullRequest, map[string][]string, error) {
	db := r.getDB(ctx)

	var prReviewers []domain.PRReviewer
	err := db.Where("reviewer_id IN ?", reviewerIDs).Find(&prReviewers).Error
	if err != nil {
		return nil, nil, err
	}

	prIDs := make(map[string]struct{})
	for _, pr := range prReviewers {
		prIDs[pr.PullRequestID] = struct{}{}
	}

	prIDList := make([]string, 0, len(prIDs))
	for prID := range prIDs {
		prIDList = append(prIDList, prID)
	}

	var prs []domain.PullRequest
	err = db.Where("pull_request_id IN ? AND status = ?", prIDList, domain.PRStatusOpen).Find(&prs).Error
	if err != nil {
		return nil, nil, err
	}

	reviewersMap := make(map[string][]string)
	for _, pr := range prs {
		reviewers, err := r.GetPRReviewers(ctx, pr.PullRequestID)
		if err != nil {
			return nil, nil, err
		}
		reviewersMap[pr.PullRequestID] = reviewers
	}

	return prs, reviewersMap, nil
}

func (r *PostgresRepository) BulkReassignReviewers(ctx context.Context, reassignments []domain.PRReassignment) error {
	db := r.getDB(ctx)

	for _, reassignment := range reassignments {
		if err := db.Where("pull_request_id = ? AND reviewer_id = ?",
			reassignment.PullRequestID, reassignment.OldReviewerID).
			Delete(&domain.PRReviewer{}).Error; err != nil {
			return err
		}

		if reassignment.NewReviewerID != "" {
			var count int64
			if err := db.Model(&domain.PRReviewer{}).
				Where("pull_request_id = ? AND reviewer_id = ?",
					reassignment.PullRequestID, reassignment.NewReviewerID).
				Count(&count).Error; err != nil {
				return domain.NewDatabaseError("check existing reviewer", err)
			}

			if count == 0 {
				newPRReviewer := domain.PRReviewer{
					PullRequestID: reassignment.PullRequestID,
					ReviewerID:    reassignment.NewReviewerID,
				}
				if err := db.Create(&newPRReviewer).Error; err != nil {
					return domain.NewDatabaseError("add new reviewer", err)
				}
			}
		}
	}

	return nil
}

func (r *PostgresRepository) GetAssignmentStats(ctx context.Context) (map[string]int, error) {
	db := r.getDB(ctx)

	var results []struct {
		ReviewerID string
		Count      int
	}

	err := db.Model(&domain.PRReviewer{}).
		Select("reviewer_id, COUNT(*) as count").
		Group("reviewer_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	for _, result := range results {
		stats[result.ReviewerID] = result.Count
	}

	return stats, nil
}
