package analytics

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fitbase/internal/models"
	"time"
)

type AnalyticsFilterQuery struct {
	From     string `form:"from"`
	To       string `form:"to"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (q *AnalyticsFilterQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 10
	}
}

type ManagerShiftReport struct {
	ShiftID        uint64     `json:"shift_id" gorm:"column:shift_id"`
	GymID          uint64     `json:"gym_id" gorm:"column:gym_id"`
	GymName        string     `json:"gym_name" gorm:"column:gym_name"`
	ManagerAuthID  uint64     `json:"manager_auth_id" gorm:"column:manager_auth_id"`
	ManagerName    string     `json:"manager_name" gorm:"column:manager_name"`
	ManagerEmail   string     `json:"manager_email" gorm:"column:manager_email"`
	OpenTime       time.Time  `json:"open_time" gorm:"column:open_time"`
	CloseTime      *time.Time `json:"close_time" gorm:"column:close_time"`
	OpenCash       float64    `json:"open_cash" gorm:"column:open_cash"`
	CloseCash      *float64   `json:"close_cash" gorm:"column:close_cash"`
	TotalCollected float64    `json:"total_collected" gorm:"column:total_collected"`
	Difference     *float64   `json:"difference" gorm:"column:difference"`
	Status         string     `json:"status" gorm:"column:status"`
}

type PaginatedManagerShifts struct {
	Data     []ManagerShiftReport `json:"data"`
	Paginate models.Pagination    `json:"paginate"`
}

type HourlyAttendanceItem struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

type HourlyDistributionList []HourlyAttendanceItem

func (l HourlyDistributionList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *HourlyDistributionList) Scan(value any) error {
	if value == nil {
		*l = HourlyDistributionList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = HourlyDistributionList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = HourlyDistributionList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type GymAttendanceReport struct {
	GymID              uint64                 `json:"gym_id" gorm:"column:gym_id"`
	GymName            string                 `json:"gym_name" gorm:"column:gym_name"`
	TotalCheckIns      int64                  `json:"total_check_ins" gorm:"column:total_check_ins"`
	PeakHour           int                    `json:"peak_hour" gorm:"column:peak_hour"`
	PeakHourCount      int64                  `json:"peak_hour_count" gorm:"column:peak_hour_count"`
	MembershipsCount   int64                  `json:"memberships_count" gorm:"column:memberships_count"`
	DayPassesCount     int64                  `json:"day_passes_count" gorm:"column:day_passes_count"`
	GuestsCount        int64                  `json:"guests_count" gorm:"column:guests_count"`
	HourlyDistribution HourlyDistributionList `json:"hourly_distribution" gorm:"column:hourly_distribution"`
}

type RevenueByMethodItem struct {
	PaymentMethodID   uint64  `json:"payment_method_id"`
	PaymentMethodName string  `json:"payment_method_name"`
	TotalRevenue      float64 `json:"total_revenue"`
	TransactionCount  int64   `json:"transaction_count"`
}

type RevenueByMethodList []RevenueByMethodItem

func (l RevenueByMethodList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *RevenueByMethodList) Scan(value any) error {
	if value == nil {
		*l = RevenueByMethodList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = RevenueByMethodList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = RevenueByMethodList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type RevenueByPlanItem struct {
	PlanID            uint64  `json:"plan_id"`
	PlanName          string  `json:"plan_name"`
	TotalRevenue      float64 `json:"total_revenue"`
	SubscriptionCount int64   `json:"subscription_count"`
}

type RevenueByPlanList []RevenueByPlanItem

func (l RevenueByPlanList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *RevenueByPlanList) Scan(value any) error {
	if value == nil {
		*l = RevenueByPlanList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = RevenueByPlanList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = RevenueByPlanList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type RevenueCrossedReport struct {
	GymID           uint64              `json:"gym_id" gorm:"column:gym_id"`
	GymName         string              `json:"gym_name" gorm:"column:gym_name"`
	TotalRevenue    float64             `json:"total_revenue" gorm:"column:total_revenue"`
	ByPaymentMethod RevenueByMethodList `json:"by_payment_method" gorm:"column:by_payment_method"`
	ByPlan          RevenueByPlanList   `json:"by_plan" gorm:"column:by_plan"`
}

type ClientSubscriptionItem struct {
	ID        uint64 `json:"id"`
	PlanName  string `json:"plan_name"`
	Status    string `json:"status"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type ClientSubscriptionList []ClientSubscriptionItem

func (l ClientSubscriptionList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *ClientSubscriptionList) Scan(value any) error {
	if value == nil {
		*l = ClientSubscriptionList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = ClientSubscriptionList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = ClientSubscriptionList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type ClientPaymentItem struct {
	ID     uint64    `json:"id"`
	Amount float64   `json:"amount"`
	Method string    `json:"method"`
	Status string    `json:"status"`
	Date   time.Time `json:"date"`
}

type ClientPaymentList []ClientPaymentItem

func (l ClientPaymentList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *ClientPaymentList) Scan(value any) error {
	if value == nil {
		*l = ClientPaymentList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = ClientPaymentList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = ClientPaymentList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type ClientCheckInItem struct {
	ID         uint64    `json:"id"`
	AccessType string    `json:"access_type"`
	Date       time.Time `json:"date"`
}

type ClientCheckInList []ClientCheckInItem

func (l ClientCheckInList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *ClientCheckInList) Scan(value any) error {
	if value == nil {
		*l = ClientCheckInList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = ClientCheckInList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = ClientCheckInList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type ClientMembershipHistoryReport struct {
	UserID         uint64                 `json:"user_id" gorm:"column:user_id"`
	FullName       string                 `json:"full_name" gorm:"column:full_name"`
	Email          string                 `json:"email" gorm:"column:email"`
	Phone          string                 `json:"phone" gorm:"column:phone"`
	TotalSpent     float64                `json:"total_spent" gorm:"column:total_spent"`
	TotalCheckIns  int64                  `json:"total_check_ins" gorm:"column:total_check_ins"`
	Subscriptions  ClientSubscriptionList `json:"subscriptions" gorm:"column:subscriptions"`
	Payments       ClientPaymentList      `json:"payments" gorm:"column:payments"`
	RecentCheckIns ClientCheckInList      `json:"recent_check_ins" gorm:"column:recent_check_ins"`
}

type DashboardKPIs struct {
	TotalRevenue      float64 `json:"total_revenue"`
	TotalTransactions int64   `json:"total_transactions"`
	ActiveSubscribers int64   `json:"active_subscribers"`
	TotalCheckIns     int64   `json:"total_check_ins"`
	PeakHour          int     `json:"peak_hour"`
	PeakHourCount     int64   `json:"peak_hour_count"`
}

func (k DashboardKPIs) Value() (driver.Value, error) {
	return json.Marshal(k)
}

func (k *DashboardKPIs) Scan(value any) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, k)
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), k)
	}
	return nil
}

type DashboardRecentPayment struct {
	ID                uint64    `json:"id"`
	Amount            float64   `json:"amount"`
	PaymentMethodName string    `json:"payment_method_name"`
	PaymentStatusName string    `json:"payment_status_name"`
	PaymentDate       time.Time `json:"payment_date"`
}

type DashboardRecentPaymentList []DashboardRecentPayment

func (l DashboardRecentPaymentList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *DashboardRecentPaymentList) Scan(value any) error {
	if value == nil {
		*l = DashboardRecentPaymentList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = DashboardRecentPaymentList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = DashboardRecentPaymentList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type DashboardRecentShift struct {
	ShiftID     uint64     `json:"shift_id"`
	ManagerName string     `json:"manager_name"`
	OpenTime    time.Time  `json:"open_time"`
	CloseTime   *time.Time `json:"close_time"`
	OpenCash    float64    `json:"open_cash"`
	Status      string     `json:"status"`
}

type DashboardRecentShiftList []DashboardRecentShift

func (l DashboardRecentShiftList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return json.Marshal(l)
}

func (l *DashboardRecentShiftList) Scan(value any) error {
	if value == nil {
		*l = DashboardRecentShiftList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*l = DashboardRecentShiftList{}
			return nil
		}
		return json.Unmarshal(v, l)
	case string:
		if v == "" {
			*l = DashboardRecentShiftList{}
			return nil
		}
		return json.Unmarshal([]byte(v), l)
	}
	return nil
}

type DashboardSummaryReport struct {
	GymID          uint64                     `json:"gym_id" gorm:"column:gym_id"`
	GymName        string                     `json:"gym_name" gorm:"column:gym_name"`
	KPIs           DashboardKPIs              `json:"kpis" gorm:"column:kpis"`
	RecentPayments DashboardRecentPaymentList `json:"recent_payments" gorm:"column:recent_payments"`
	RecentShifts   DashboardRecentShiftList   `json:"recent_shifts" gorm:"column:recent_shifts"`
}

type Repository interface {
	GetManagerShiftsReport(ctx context.Context, gymID uint64, from, to time.Time, offset, limit int) ([]ManagerShiftReport, int64, error)
	GetGymAttendanceReport(ctx context.Context, gymID uint64, from, to time.Time) (*GymAttendanceReport, error)
	GetRevenueCrossedReport(ctx context.Context, gymID uint64, from, to time.Time) (*RevenueCrossedReport, error)
	GetClientHistoryReport(ctx context.Context, gymID, targetUserID uint64) (*ClientMembershipHistoryReport, error)
	GetDashboardSummary(ctx context.Context, gymID uint64) (*DashboardSummaryReport, error)
	GetGymOwnerID(ctx context.Context, gymID uint64) (uint64, error)
	GetOwnerIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
}

type Service interface {
	GetManagerShifts(ctx context.Context, authID, gymID uint64, fromStr, toStr string, page, pageSize int) (*PaginatedManagerShifts, error)
	GetAttendanceMetrics(ctx context.Context, authID, gymID uint64, fromStr, toStr string) (*GymAttendanceReport, error)
	GetCrossedRevenue(ctx context.Context, authID, gymID uint64, fromStr, toStr string) (*RevenueCrossedReport, error)
	GetClientHistory(ctx context.Context, authID, gymID, targetUserID uint64) (*ClientMembershipHistoryReport, error)
	GetDashboardSummary(ctx context.Context, authID, gymID uint64) (*DashboardSummaryReport, error)
}
