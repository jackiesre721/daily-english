package repository

import (
	"database/sql"
	"daily-english/internal/model"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Get() (*model.User, error) {
	var u model.User
	var lastCheckin sql.NullString
	err := r.db.QueryRow(`SELECT id, name, streak_days, max_streak_days, freeze_cards, daily_new_target, total_words_learned, last_checkin_date, settings_json, created_at, updated_at FROM users WHERE id = 'user_001'`).
		Scan(&u.ID, &u.Name, &u.StreakDays, &u.MaxStreakDays, &u.FreezeCards, &u.DailyNewTarget, &u.TotalWordsLearned, &lastCheckin, &u.SettingsJSON, &u.CreatedAt, &u.UpdatedAt)
	if lastCheckin.Valid {
		u.LastCheckinDate = lastCheckin.String
	}
	return &u, err
}

func (r *UserRepo) Update(u *model.User) error {
	_, err := r.db.Exec(`UPDATE users SET name=?, streak_days=?, max_streak_days=?, freeze_cards=?, daily_new_target=?, total_words_learned=?, last_checkin_date=?, settings_json=?, updated_at=datetime('now', 'localtime') WHERE id=?`,
		u.Name, u.StreakDays, u.MaxStreakDays, u.FreezeCards, u.DailyNewTarget, u.TotalWordsLearned, u.LastCheckinDate, u.SettingsJSON, u.ID)
	return err
}

// AtomicCheckin atomically updates streak and checkin state to avoid race conditions.
func (r *UserRepo) AtomicCheckin(today, yesterday string, streakBroken bool, useFreezeCard bool, awardFreezeCard bool) (*model.User, error) {
	tx, err := r.db.BeginTx()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Lock row for update
	var u model.User
	var lastCheckin sql.NullString
	err = tx.QueryRow(`SELECT id, name, streak_days, max_streak_days, freeze_cards, daily_new_target, total_words_learned, last_checkin_date, settings_json, created_at, updated_at FROM users WHERE id = 'user_001'`).
		Scan(&u.ID, &u.Name, &u.StreakDays, &u.MaxStreakDays, &u.FreezeCards, &u.DailyNewTarget, &u.TotalWordsLearned, &lastCheckin, &u.SettingsJSON, &u.CreatedAt, &u.UpdatedAt)
	if lastCheckin.Valid {
		u.LastCheckinDate = lastCheckin.String
	}
	if err != nil {
		return nil, err
	}

	if streakBroken {
		if useFreezeCard {
			u.FreezeCards--
			u.StreakDays++
		} else {
			u.StreakDays = 1
		}
	} else {
		u.StreakDays++
	}

	if u.StreakDays > u.MaxStreakDays {
		u.MaxStreakDays = u.StreakDays
	}
	if awardFreezeCard {
		u.FreezeCards++
	}
	u.LastCheckinDate = today

	_, err = tx.Exec(`UPDATE users SET streak_days=?, max_streak_days=?, freeze_cards=?, last_checkin_date=?, updated_at=datetime('now', 'localtime') WHERE id=?`,
		u.StreakDays, u.MaxStreakDays, u.FreezeCards, u.LastCheckinDate, u.ID)
	if err != nil {
		return nil, err
	}

	return &u, tx.Commit()
}

func (r *UserRepo) ResetProgress() error {
	_, err := r.db.Exec(`UPDATE users SET streak_days=0, max_streak_days=0, freeze_cards=0, total_words_learned=0, last_checkin_date=NULL, updated_at=datetime('now', 'localtime') WHERE id='user_001'`)
	return err
}
