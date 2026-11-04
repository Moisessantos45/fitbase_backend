package analytics

import (
	"context"
	"fitbase/internal/models"
	"fitbase/internal/shared/errorsx"
	"fitbase/internal/shared/helper"
	"time"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetManagerShiftsReport(
	ctx context.Context,
	gymID uint64,
	from, to time.Time,
	offset, limit int,
) ([]ManagerShiftReport, int64, error) {
	base := r.db.WithContext(ctx).
		Table("cash_registers").
		Where("cash_registers.gym_id = ?", gymID)

	if !from.IsZero() {
		base = base.Where("cash_registers.open_time >= ?", from)
	}

	if !to.IsZero() {
		base = base.Where("cash_registers.open_time <= ?", to)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).
		Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	query := `
		WITH collected AS (
			SELECT
				cash_register_id,
				SUM(amount) AS total_cash
			FROM payments
			WHERE payment_status_id = ?
			GROUP BY cash_register_id
		)
		SELECT
			cr.id AS shift_id,
			cr.gym_id,
			g.name AS gym_name,
			cr.auth_id AS manager_auth_id,
			COALESCE(
				NULLIF(TRIM(CONCAT_WS(' ', gm.first_name, gm.last_name)), ''),
				a.email
			) AS manager_name,
			a.email AS manager_email,
			cr.open_time,
			cr.close_time,
			cr.open_cash,
			cr.close_cash,
			COALESCE(c.total_cash, 0) AS total_collected,
			CASE
				WHEN cr.close_time IS NULL THEN NULL
				ELSE ROUND(
					(cr.close_cash - (cr.open_cash + COALESCE(c.total_cash, 0)))::numeric,
					2
				)
			END AS difference,
			CASE
				WHEN cr.close_time IS NULL THEN 'Abierta'
				WHEN ABS(
					cr.close_cash - (cr.open_cash + COALESCE(c.total_cash, 0))
				) < 0.01 THEN 'Cuadrada'
				ELSE 'Con Descuadre'
			END AS status
		FROM cash_registers cr
		JOIN gyms g ON g.id = cr.gym_id
		JOIN auths a ON a.id = cr.auth_id
		LEFT JOIN gym_managers gm ON gm.auth_id = cr.auth_id
		LEFT JOIN collected c ON c.cash_register_id = cr.id
		WHERE cr.gym_id = ?
	`

	args := []any{
		models.PaymentStatusPaidID,
		gymID,
	}

	if !from.IsZero() {
		query += " AND cr.open_time >= ?"
		args = append(args, from)
	}

	if !to.IsZero() {
		query += " AND cr.open_time <= ?"
		args = append(args, to)
	}

	query += `
		ORDER BY cr.open_time DESC
		OFFSET ? LIMIT ?
	`

	args = append(args, offset, limit)

	var results []ManagerShiftReport
	if err := r.db.WithContext(ctx).
		Raw(query, args...).
		Scan(&results).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return results, total, nil
}

func (r *PostgresRepository) GetGymAttendanceReport(
	ctx context.Context, gymID uint64, from, to time.Time,
) (*GymAttendanceReport, error) {
	dateFilter := ""
	var args []any
	args = append(args, models.AccessTypeMembershipID, models.AccessTypeDayPassID, models.AccessTypeGuestID, gymID)

	if !from.IsZero() {
		dateFilter += " AND ci.checked_in_at >= ?"
		args = append(args, from)
	}
	if !to.IsZero() {
		dateFilter += " AND ci.checked_in_at <= ?"
		args = append(args, to)
	}

	mainCheckInDateFilter := ""
	if !from.IsZero() {
		mainCheckInDateFilter += " AND check_ins.checked_in_at >= ?"
		args = append(args, from)
	}
	if !to.IsZero() {
		mainCheckInDateFilter += " AND check_ins.checked_in_at <= ?"
		args = append(args, to)
	}

	peakDateFilter := ""
	args = append(args, gymID)
	if !from.IsZero() {
		peakDateFilter += " AND ci.checked_in_at >= ?"
		args = append(args, from)
	}
	if !to.IsZero() {
		peakDateFilter += " AND ci.checked_in_at <= ?"
		args = append(args, to)
	}

	query := `
		SELECT
			gyms.id AS gym_id,
			gyms.name AS gym_name,
			COUNT(check_ins.id) AS total_check_ins,
			COALESCE(peak.hr, 0) AS peak_hour,
			COALESCE(peak.cnt, 0) AS peak_hour_count,
			COUNT(CASE WHEN check_ins.access_type_id = ? THEN 1 END) AS memberships_count,
			COUNT(CASE WHEN check_ins.access_type_id = ? THEN 1 END) AS day_passes_count,
			COUNT(CASE WHEN check_ins.access_type_id = ? THEN 1 END) AS guests_count,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object('hour', h.hr, 'count', h.cnt) ORDER BY h.hr ASC)
				FROM (
					SELECT EXTRACT(HOUR FROM ci.checked_in_at)::int AS hr, COUNT(ci.id) AS cnt
					FROM check_ins ci
					WHERE ci.gym_id = ?` + dateFilter + `
					GROUP BY EXTRACT(HOUR FROM ci.checked_in_at)::int
				) h
			), '[]'::jsonb) AS hourly_distribution
		FROM gyms
		LEFT JOIN check_ins ON check_ins.gym_id = gyms.id` + mainCheckInDateFilter + `
		LEFT JOIN (
			SELECT EXTRACT(HOUR FROM ci.checked_in_at)::int AS hr, COUNT(ci.id) AS cnt
			FROM check_ins ci
			WHERE ci.gym_id = ?` + peakDateFilter + `
			GROUP BY EXTRACT(HOUR FROM ci.checked_in_at)::int
			ORDER BY cnt DESC, hr ASC
			LIMIT 1
		) peak ON true
		WHERE gyms.id = ?
		GROUP BY gyms.id, gyms.name, peak.hr, peak.cnt
	`
	args = append(args, gymID)

	var report GymAttendanceReport
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&report).Error; err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if report.GymID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &report, nil
}

func (r *PostgresRepository) GetRevenueCrossedReport(ctx context.Context, gymID uint64, from, to time.Time) (*RevenueCrossedReport, error) {
	methodDateFilter := ""
	var methodArgs []any
	methodArgs = append(methodArgs, gymID, models.PaymentStatusPaidID)
	if !from.IsZero() {
		methodDateFilter += " AND p.created_at >= ?"
		methodArgs = append(methodArgs, from)
	}
	if !to.IsZero() {
		methodDateFilter += " AND p.created_at <= ?"
		methodArgs = append(methodArgs, to)
	}

	planDateFilter := ""
	var planArgs []any
	planArgs = append(planArgs, gymID, models.PaymentStatusPaidID)
	if !from.IsZero() {
		planDateFilter += " AND p.created_at >= ?"
		planArgs = append(planArgs, from)
	}
	if !to.IsZero() {
		planDateFilter += " AND p.created_at <= ?"
		planArgs = append(planArgs, to)
	}

	query := `
		SELECT
			gyms.id AS gym_id,
			gyms.name AS gym_name,
			COALESCE(SUM(payments.amount), 0) AS total_revenue,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'payment_method_id', pm.id,
					'payment_method_name', pm.name,
					'total_revenue', COALESCE(agg.sum_amount, 0),
					'transaction_count', COALESCE(agg.cnt, 0)
				))
				FROM payment_methods pm
				JOIN (
					SELECT p.payment_method_id, SUM(p.amount) AS sum_amount, COUNT(p.id) AS cnt
					FROM payments p
					WHERE p.gym_id = ? AND p.payment_status_id = ?` + methodDateFilter + `
					GROUP BY p.payment_method_id
				) agg ON agg.payment_method_id = pm.id
			), '[]'::jsonb) AS by_payment_method,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'plan_id', pl.id,
					'plan_name', pl.name,
					'total_revenue', COALESCE(agg.sum_amount, 0),
					'subscription_count', COALESCE(agg.cnt, 0)
				))
				FROM plans pl
				JOIN (
					SELECT s.plan_id, SUM(p.amount) AS sum_amount, COUNT(p.id) AS cnt
					FROM payments p
					JOIN subscriptions s ON s.id = p.subscription_id
					WHERE p.gym_id = ? AND p.payment_status_id = ?` + planDateFilter + `
					GROUP BY s.plan_id
				) agg ON agg.plan_id = pl.id
			), '[]'::jsonb) AS by_plan
		FROM gyms
		LEFT JOIN payments ON payments.gym_id = gyms.id AND payments.payment_status_id = ?`

	var allArgs []any
	allArgs = append(allArgs, methodArgs...)
	allArgs = append(allArgs, planArgs...)
	allArgs = append(allArgs, models.PaymentStatusPaidID)

	if !from.IsZero() {
		query += " AND payments.created_at >= ?"
		allArgs = append(allArgs, from)
	}
	if !to.IsZero() {
		query += " AND payments.created_at <= ?"
		allArgs = append(allArgs, to)
	}

	query += `
		WHERE gyms.id = ?
		GROUP BY gyms.id, gyms.name
	`
	allArgs = append(allArgs, gymID)

	var report RevenueCrossedReport
	if err := r.db.WithContext(ctx).Raw(query, allArgs...).Scan(&report).Error; err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if report.GymID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &report, nil
}

func (r *PostgresRepository) GetClientHistoryReport(ctx context.Context, gymID, targetUserID uint64) (*ClientMembershipHistoryReport, error) {
	query := `
		SELECT
			users.id AS user_id,
			CONCAT(users.first_name, ' ', users.last_name) AS full_name,
			auths.email,
			users.phone,
			COALESCE((
				SELECT SUM(p.amount)
				FROM payments p
				LEFT JOIN subscriptions s ON s.id = p.subscription_id
				LEFT JOIN day_passes dp ON dp.id = p.day_pass_id
				WHERE p.gym_id = ?
				  AND p.payment_status_id = ?
				  AND (s.user_id = users.id OR dp.user_id = users.id)
			), 0) AS total_spent,
			COALESCE((
				SELECT COUNT(ci.id)
				FROM check_ins ci
				WHERE ci.gym_id = ? AND ci.user_id = users.id
			), 0) AS total_check_ins,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'id', sub.id,
					'plan_name', plans.name,
					'status', status_subscriptions.name,
					'start_date', to_char(sub.start_date, 'YYYY-MM-DD'),
					'end_date', to_char(sub.end_date, 'YYYY-MM-DD')
				) ORDER BY sub.created_at DESC)
				FROM subscriptions sub
				JOIN plans ON plans.id = sub.plan_id
				JOIN status_subscriptions ON status_subscriptions.id = sub.status_id
				WHERE sub.user_id = users.id AND plans.gym_id = ?
			), '[]'::jsonb) AS subscriptions,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'id', p.id,
					'amount', p.amount,
					'method', payment_methods.name,
					'status', payment_statuses.name,
					'date', p.payment_date
				) ORDER BY p.created_at DESC)
				FROM payments p
				JOIN payment_methods ON payment_methods.id = p.payment_method_id
				JOIN payment_statuses ON payment_statuses.id = p.payment_status_id
				LEFT JOIN subscriptions s ON s.id = p.subscription_id
				LEFT JOIN day_passes dp ON dp.id = p.day_pass_id
				WHERE p.gym_id = ? AND (s.user_id = users.id OR dp.user_id = users.id)
			), '[]'::jsonb) AS payments,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'id', ci.id,
					'access_type', access_types.name,
					'date', ci.checked_in_at
				) ORDER BY ci.checked_in_at DESC)
				FROM (
					SELECT ci_sub.id, ci_sub.access_type_id, ci_sub.checked_in_at
					FROM check_ins ci_sub
					WHERE ci_sub.gym_id = ? AND ci_sub.user_id = users.id
					ORDER BY ci_sub.checked_in_at DESC
					LIMIT 10
				) ci
				JOIN access_types ON access_types.id = ci.access_type_id
			), '[]'::jsonb) AS recent_check_ins
		FROM users
		JOIN auths ON auths.id = users.auth_id
		WHERE users.id = ?
	`

	var report ClientMembershipHistoryReport
	err := r.db.WithContext(ctx).Raw(query,
		gymID, models.PaymentStatusPaidID,
		gymID,
		gymID,
		gymID,
		gymID,
		targetUserID,
	).Scan(&report).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if report.UserID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &report, nil
}

func (r *PostgresRepository) GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error) {
	var ownerID uint64
	err := r.db.WithContext(ctx).
		Model(&models.Gym{}).
		Select("owner_id").
		Where("id = ?", gymID).
		Scan(&ownerID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if ownerID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return ownerID, nil
}

func (r *PostgresRepository) GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var ownerID uint64
	err := r.db.WithContext(ctx).
		Model(&models.Owner{}).
		Select("id").
		Where("auth_id = ?", authID).
		Scan(&ownerID).Error

	if err != nil {
		return 0, helper.TranslateDBError(err)
	}

	if ownerID == 0 {
		return 0, errorsx.ErrNotFound
	}

	return ownerID, nil
}

func (r *PostgresRepository) GetDashboardSummary(ctx context.Context, gymID uint64) (*DashboardSummaryReport, error) {
	query := `
		SELECT
			g.id AS gym_id,
			g.name AS gym_name,
			jsonb_build_object(
				'total_revenue', COALESCE(rev.total_revenue, 0),
				'total_transactions', COALESCE(rev.total_transactions, 0),
				'active_subscribers', COALESCE(sub.active_subscribers, 0),
				'total_check_ins', COALESCE(chk.total_check_ins, 0),
				'peak_hour', COALESCE(pk.peak_hour, 0),
				'peak_hour_count', COALESCE(pk.peak_hour_count, 0)
			) AS kpis,
			COALESCE((
				SELECT jsonb_agg(
					jsonb_build_object(
						'id', p.id,
						'amount', p.amount,
						'payment_method_name', COALESCE(pm.name, 'Efectivo'),
						'payment_status_name', COALESCE(ps.name, 'Pagado'),
						'payment_date', COALESCE(p.payment_date, p.created_at)
					)
				)
				FROM (
					SELECT p_sub.*
					FROM payments p_sub
					WHERE p_sub.gym_id = g.id
					ORDER BY p_sub.created_at DESC
					LIMIT 5
				) p
				LEFT JOIN payment_methods pm ON pm.id = p.payment_method_id
				LEFT JOIN payment_statuses ps ON ps.id = p.payment_status_id
			), '[]'::jsonb) AS recent_payments,
			COALESCE((
				SELECT jsonb_agg(
					jsonb_build_object(
						'shift_id', cr.id,
						'manager_name', COALESCE(NULLIF(TRIM(CONCAT_WS(' ', gm.first_name, gm.last_name)), ''), a.email),
						'open_time', cr.open_time,
						'close_time', cr.close_time,
						'open_cash', cr.open_cash,
						'status', CASE
							WHEN cr.close_time IS NULL THEN 'Abierta'
							WHEN ABS(cr.close_cash - (cr.open_cash + COALESCE(col.total_cash, 0))) < 0.01 THEN 'Cuadrada'
							ELSE 'Con Descuadre'
						END
					)
				)
				FROM (
					SELECT cr_sub.*
					FROM cash_registers cr_sub
					WHERE cr_sub.gym_id = g.id
					ORDER BY cr_sub.open_time DESC
					LIMIT 5
				) cr
				JOIN auths a ON a.id = cr.auth_id
				LEFT JOIN gym_managers gm ON gm.auth_id = cr.auth_id
				LEFT JOIN (
					SELECT cash_register_id, SUM(amount) AS total_cash
					FROM payments
					WHERE payment_status_id = ?
					GROUP BY cash_register_id
				) col ON col.cash_register_id = cr.id
			), '[]'::jsonb) AS recent_shifts
		FROM gyms g
		LEFT JOIN (
			SELECT gym_id, COALESCE(SUM(amount), 0) AS total_revenue, COUNT(id) AS total_transactions
			FROM payments
			WHERE payment_status_id = ?
			GROUP BY gym_id
		) rev ON rev.gym_id = g.id
		LEFT JOIN (
			SELECT pl.gym_id, COUNT(s.id) AS active_subscribers
			FROM subscriptions s
			JOIN plans pl ON pl.id = s.plan_id
			WHERE s.status_id = ? AND s.end_date >= CURRENT_DATE
			GROUP BY pl.gym_id
		) sub ON sub.gym_id = g.id
		LEFT JOIN (
			SELECT gym_id, COUNT(id) AS total_check_ins
			FROM check_ins
			GROUP BY gym_id
		) chk ON chk.gym_id = g.id
		LEFT JOIN (
			SELECT ci.gym_id, EXTRACT(HOUR FROM ci.checked_in_at)::int AS peak_hour, COUNT(ci.id) AS peak_hour_count
			FROM check_ins ci
			WHERE ci.gym_id = ?
			GROUP BY ci.gym_id, EXTRACT(HOUR FROM ci.checked_in_at)::int
			ORDER BY COUNT(ci.id) DESC, peak_hour ASC
			LIMIT 1
		) pk ON pk.gym_id = g.id
		WHERE g.id = ?
	`

	args := []any{
		models.PaymentStatusPaidID,
		models.PaymentStatusPaidID,
		models.StatusSubscriptionActiveID,
		gymID,
		gymID,
	}

	var report DashboardSummaryReport
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&report).Error
	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	if report.GymID == 0 {
		return nil, errorsx.ErrNotFound
	}

	return &report, nil
}
