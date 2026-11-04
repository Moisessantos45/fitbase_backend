package models

import (
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Auth struct {
	ID               uint64 `json:"id" gorm:"primaryKey"`
	RoleID           uint64 `json:"role_id" gorm:"not null;index"`
	Email            string `json:"email" gorm:"size:150;uniqueIndex;not null"`
	PasswordHash     string `json:"-" gorm:"size:255;not null"`
	TwoFactorSecret  string `json:"-" gorm:"size:26"`
	TwoFactorEnabled bool   `json:"two_factor_enabled" gorm:"default:false"`
	FirstSession     bool   `json:"first_session" gorm:"default:true"`
	FullProfile      bool   `json:"full_profile" gorm:"default:false"`
	EmailConfirmed   bool   `json:"email_confirmed" gorm:"default:false"`
	Active           bool   `json:"active" gorm:"default:true"`
	Token            string `json:"token" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Role Role `json:"-" gorm:"foreignKey:RoleID"`
}

type File struct {
	ID       uint64 `json:"id" gorm:"primaryKey"`
	Slug     string `json:"slug" gorm:"unique;index"`
	Path     string `json:"-" gorm:"type:text;not null"`
	MimeType string `json:"mime_type" gorm:"not null"` // "image/jpeg",
	Size     int64  `json:"size" gorm:"not null"`      // en bytes}
	AuthorID uint64 `json:"author_id" gorm:"not null;index"`

	Url string `json:"url" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Author Auth `json:"-" gorm:"foreignKey:AuthorID"`
}

type UploadedFile struct {
	File     *multipart.FileHeader
	Size     int64
	MimeType string
}

type User struct {
	ID        uint64  `json:"id" gorm:"primaryKey"`
	AuthID    uint64  `json:"auth_id" gorm:"not null;uniqueIndex"`
	FirstName string  `json:"first_name" gorm:"size:100;not null"`
	LastName  string  `json:"last_name" gorm:"size:100;not null"`
	Phone     string  `json:"phone" gorm:"size:20;not null"`
	AvatarID  *uint64 `json:"avatar_id" gorm:"index"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Auth   Auth  `json:"-" gorm:"foreignKey:AuthID"`
	Avatar *File `json:"avatar,omitempty" gorm:"foreignKey:AvatarID"`
}

type Owner struct {
	ID        uint64  `json:"id" gorm:"primaryKey"`
	AuthID    uint64  `json:"auth_id" gorm:"not null;uniqueIndex"`
	FirstName string  `json:"first_name" gorm:"size:100;not null"`
	LastName  string  `json:"last_name" gorm:"size:100;not null"`
	Phone     string  `json:"phone" gorm:"size:20;not null"`
	AvatarID  *uint64 `json:"avatar_id" gorm:"index"`

	AvatarURL string `json:"avatar_url" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Auth   Auth  `json:"-" gorm:"foreignKey:AuthID"`
	Avatar *File `json:"avatar,omitempty" gorm:"foreignKey:AvatarID"`
	Gyms   []Gym `json:"gyms,omitempty" gorm:"foreignKey:OwnerID"`
}

type Gym struct {
	ID           uint64  `json:"id" gorm:"primaryKey"`
	OwnerID      uint64  `json:"owner_id" gorm:"not null;index"`
	BackgroundID *uint64 `json:"background_id" gorm:"index"`
	Name         string  `json:"name" gorm:"size:100;not null"`
	Address      string  `json:"address" gorm:"size:255;not null"`
	Phone        string  `json:"phone" gorm:"size:20;not null"`
	Email        string  `json:"email" gorm:"size:100;not null"`
	Latitude     float64 `json:"latitude" gorm:"type:numeric(9,6);not null"`
	Longitude    float64 `json:"longitude" gorm:"type:numeric(9,6);not null"`
	Location     string  `json:"-" gorm:"->:false;<-:false;column:location;type:geography(Point,4326)"`

	BackgroundURL string `json:"background_url" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Owner        Owner         `json:"-" gorm:"foreignKey:OwnerID"`
	Background   *File         `json:"background,omitempty" gorm:"foreignKey:BackgroundID"`
	GymSchedules []GymSchedule `json:"gym_schedules,omitempty" gorm:"foreignKey:GymID"`
	GymPhotos    []GymPhoto    `json:"gym_photos,omitempty" gorm:"foreignKey:GymID"`
}

type GymManager struct {
	ID              uint64  `json:"id" gorm:"primaryKey"`
	GymID           uint64  `json:"gym_id" gorm:"not null;index"`
	AuthID          uint64  `json:"auth_id" gorm:"not null;uniqueIndex"`
	CreatedByAuthID uint64  `json:"created_by_auth_id" gorm:"not null"`
	FirstName       string  `json:"first_name" gorm:"size:100;not null"`
	LastName        string  `json:"last_name" gorm:"size:100;not null"`
	Phone           string  `json:"phone" gorm:"size:20;not null"`
	AvatarID        *uint64 `json:"avatar_id" gorm:"index"`

	AvatarURL string `json:"avatar_url" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym           Gym   `json:"-" gorm:"foreignKey:GymID"`
	Auth          Auth  `json:"-" gorm:"foreignKey:AuthID"`
	Avatar        *File `json:"avatar,omitempty" gorm:"foreignKey:AvatarID"`
	CreatedByAuth Auth  `json:"-" gorm:"foreignKey:CreatedByAuthID"`
}

type GymPhoto struct {
	ID     uint64 `json:"id" gorm:"primaryKey"`
	FileID uint64 `json:"file_id" gorm:"not null;index"`
	GymID  uint64 `json:"gym_id" gorm:"not null;index"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym  Gym  `json:"-" gorm:"foreignKey:GymID"`
	File File `json:"-" gorm:"foreignKey:FileID"`
}

type GymSchedule struct {
	ID           uint64 `json:"id" gorm:"primaryKey"`
	GymID        uint64 `json:"gym_id" gorm:"not null;index"`
	DayOfWeek    int    `json:"day_of_week" gorm:"not null"`
	OpenMinutes  int    `json:"open_minutes" gorm:"not null"`  // Ej: 480 (08:00)
	CloseMinutes int    `json:"close_minutes" gorm:"not null"` // Ej: 1200 (20:00)
	IsClosed     bool   `json:"is_closed" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym Gym `json:"-" gorm:"foreignKey:GymID"`
}

type CashRegister struct {
	ID        uint64     `json:"id" gorm:"primaryKey"`
	GymID     uint64     `json:"gym_id" gorm:"not null;index"`
	AuthID    uint64     `json:"auth_id" gorm:"not null;index"`
	OpenTime  time.Time  `json:"open_time" gorm:"not null"`
	CloseTime *time.Time `json:"close_time" gorm:"default:null"`
	OpenCash  float64    `json:"open_cash" gorm:"type:numeric(10,2);not null"`
	CloseCash float64    `json:"close_cash" gorm:"type:numeric(10,2)"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym  Gym  `json:"-" gorm:"foreignKey:GymID"`
	Auth Auth `json:"-" gorm:"foreignKey:AuthID"`
}

type PaymentMethod struct {
	ID     uint64 `json:"id" gorm:"primaryKey"`
	Name   string `json:"name" gorm:"size:100;not null"`
	Active bool   `json:"active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Plan struct {
	ID          uint64  `json:"id" gorm:"primaryKey"`
	GymID       uint64  `json:"gym_id" gorm:"not null;index"`
	Name        string  `json:"name" gorm:"size:100;not null"`
	Description string  `json:"description" gorm:"size:255;not null"`
	Price       float64 `json:"price" gorm:"type:numeric(10,2);not null"`
	Discount    float64 `json:"discount" gorm:"type:numeric(5,2);default:0"`
	Duration    int     `json:"duration" gorm:"not null"`
	PlanType    string  `json:"plan_type" gorm:"size:50;not null"`
	MaxVisits   *int    `json:"max_visits" gorm:"default:null"`
	Active      bool    `json:"active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym       Gym        `json:"-" gorm:"foreignKey:GymID"`
	PlanItems []PlanItem `json:"plan_items,omitempty" gorm:"foreignKey:PlanID"`
}

type PlanItem struct {
	ID     uint64 `json:"id" gorm:"primaryKey"`
	PlanID uint64 `json:"plan_id" gorm:"not null;index"`
	Name   string `json:"name" gorm:"size:100;not null"`

	Description string    `json:"description" gorm:"size:255;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Plan Plan `json:"-" gorm:"foreignKey:PlanID"`
}

type StatusSubscription struct {
	ID          uint64 `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Description string `json:"description" gorm:"size:255;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Subscription struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	UserID          uint64    `json:"user_id" gorm:"not null;index"`
	PlanID          uint64    `json:"plan_id" gorm:"not null;index"`
	StatusID        uint64    `json:"status_id" gorm:"not null;index"`
	Slug            uuid.UUID `json:"slug" gorm:"type:uuid;uniqueIndex;not null"`
	StartDate       time.Time `json:"start_date" gorm:"type:date;not null"`
	EndDate         time.Time `json:"end_date" gorm:"type:date;not null"`
	PaymentMethodID uint64    `json:"payment_method_id" gorm:"not null"`
	AutoRenew       bool      `json:"auto_renew" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	User          User               `json:"-" gorm:"foreignKey:UserID"`
	Plan          Plan               `json:"-" gorm:"foreignKey:PlanID"`
	PaymentMethod PaymentMethod      `json:"-" gorm:"foreignKey:PaymentMethodID"`
	Status        StatusSubscription `json:"-" gorm:"foreignKey:StatusID"`
}

// status of payment: pendiente, pagado, cancelado, reembolsado
type PaymentStatus struct {
	ID     uint64 `json:"id" gorm:"primaryKey"`
	Name   string `json:"name" gorm:"size:100;not null"`
	Active bool   `json:"active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type DayPass struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	UserID    *uint64   `json:"user_id" gorm:"index"` // Nulo si es invitado
	GymID     uint64    `json:"gym_id" gorm:"not null;index"`
	PlanID    uint64    `json:"plan_id" gorm:"not null;index"`
	Slug      uuid.UUID `json:"slug" gorm:"type:uuid;uniqueIndex;not null"`
	VisitDate time.Time `json:"visit_date" gorm:"type:date;not null"`
	FullName  string    `json:"full_name" gorm:"size:100;not null"`
	Phone     string    `json:"phone" gorm:"size:20;not null"`
	Email     string    `json:"email" gorm:"size:100;not null"`
	IsUsed    bool      `json:"is_used" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym  Gym  `json:"-" gorm:"foreignKey:GymID"`
	Plan Plan `json:"-" gorm:"foreignKey:PlanID"`
}

type Payment struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	GymID           uint64    `json:"gym_id" gorm:"not null;index"`
	SubscriptionID  *uint64   `json:"subscription_id" gorm:"index"`
	DayPassID       *uint64   `json:"day_pass_id" gorm:"index"`
	CashRegisterID  *uint64   `json:"cash_register_id" gorm:"index"`
	PaymentMethodID uint64    `json:"payment_method_id" gorm:"not null;index"`
	PaymentStatusID uint64    `json:"payment_status_id" gorm:"not null;index"`
	Amount          float64   `json:"amount" gorm:"type:numeric(10,2);not null"`
	PaymentDate     time.Time `json:"payment_date" gorm:"not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Gym           Gym           `json:"-" gorm:"foreignKey:GymID"`
	PaymentMethod PaymentMethod `json:"-" gorm:"foreignKey:PaymentMethodID"`
	Subscription  *Subscription `json:"-" gorm:"foreignKey:SubscriptionID"`
	DayPass       *DayPass      `json:"-" gorm:"foreignKey:DayPassID"`
	CashRegister  *CashRegister `json:"-" gorm:"foreignKey:CashRegisterID"`
	PaymentStatus PaymentStatus `json:"-" gorm:"foreignKey:PaymentStatusID"`
}

type AccessType struct {
	ID     uint64 `json:"id" gorm:"primaryKey"`
	Name   string `json:"name" gorm:"size:100;not null"`
	Active bool   `json:"active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CheckIn struct {
	ID             uint64  `json:"id" gorm:"primaryKey"`
	GymID          uint64  `json:"gym_id" gorm:"not null;index"`
	AccessTypeID   uint64  `json:"access_type_id" gorm:"not null;index"`
	UserID         *uint64 `json:"user_id" gorm:"index"`
	DayPassID      *uint64 `json:"day_pass_id" gorm:"index"`
	SubscriptionID *uint64 `json:"subscription_id" gorm:"index"`
	RegisteredByID *uint64 `json:"registered_by_id" gorm:"index"`

	CheckedInAt time.Time `json:"checked_in_at" gorm:"autoCreateTime;index"`

	Gym          Gym           `json:"-" gorm:"foreignKey:GymID"`
	AccessType   AccessType    `json:"-" gorm:"foreignKey:AccessTypeID"`
	User         *User         `json:"-" gorm:"foreignKey:UserID"`
	DayPass      *DayPass      `json:"-" gorm:"foreignKey:DayPassID"`
	Subscription *Subscription `json:"-" gorm:"foreignKey:SubscriptionID"`
	RegisteredBy *Auth         `json:"-" gorm:"foreignKey:RegisteredByID"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) (err error) {
	newUUID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	s.Slug = newUUID
	return nil
}

func (d *DayPass) BeforeCreate(tx *gorm.DB) (err error) {
	newUUID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	d.Slug = newUUID
	return nil
}

var Models = []any{
	&Role{},
	&Auth{},
	&File{},
	&User{},
	&Owner{},
	&Gym{},
	&GymManager{},
	&GymSchedule{},
	&GymPhoto{},
	&CashRegister{},
	&PaymentMethod{},
	&Plan{},
	&PlanItem{},
	&StatusSubscription{},
	&Subscription{},
	&PaymentStatus{},
	&DayPass{},
	&Payment{},
	&AccessType{},
	&CheckIn{},
}
