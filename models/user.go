// I removed the verify password by encryption
package models

import (
	"errors"
	"fmt"
	"time"

	goval "github.com/go-passwd/validator"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	// "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	// enTranslations "github.com/go-playground/validator/v10"
	"github.com/leebenson/conform"
	// "golang.org/x/crypto/bcrypt"
)

const (
    AdminRole = "admin"
    CIURole = "ciu"
    CRURole = "cru"
)

// User struct representing a user in the system
type User struct {
	Model
    ID        uint           `gorm:"primaryKey"`
    Name      string         `gorm:"size:255"`
	Fullname       string         `json:"fullname" binding:"required,min=2"`
	Username       string         `json:"username" binding:"required,min=2"`
	Telephone      string         `json:"telephone" gorm:"unique;default:null" binding:"required"`
    Email     string         `gorm:"unique;not null"`
    Password       string         `json:"password,omitempty" gorm:"-"`
	IsEmailActive  bool           `json:"-"`
	HashedPassword string         `json:"-"`
	AdminStatus    bool           `json:"is_admin" gorm:"foreignKey:Status"`
	ThumbNailURL   string         `json:"thumbnail_url,omitempty"`
	RoleID      uuid.UUID `gorm:"type:uuid" json:"role_id"`
	Role        Role      `gorm:"foreignKey:RoleID" json:"role"`
}

type Admin struct {
	Model
	Status bool `json:"is_admin"`
}

// CreateSocialUserParams represents the parameters required to create a new social user.
type CreateSocialUserParams struct {
	Email    string `json:"email"`
	IsSocial bool   `json:"is_social"`
	Active   bool   `json:"active"`
	Name     string `json:"name"`
}

func ValidatePassword(password string) error {
	passwordValidator := goval.New(goval.MinLength(6, errors.New("password cant be less than 6 characters")),
		goval.MaxLength(15, errors.New("password cant be more than 15 characters")))
	err := passwordValidator.Validate(password)
	return err
}
func validateWhiteSpaces(data interface{}) error {
	return conform.Strings(data)
}

func translateError(err error, trans ut.Translator) (errs []error) {
	if err == nil {
		return nil
	}
	validatorErrs := err.(validator.ValidationErrors)
	for _, e := range validatorErrs {
		translatedErr := fmt.Errorf(e.Translate(trans) + "; ")
		errs = append(errs, translatedErr)
	}
	return errs
}

type UserResponse struct {
	ID        uint   `json:"id"`
	Fullname  string `json:"fullname"`
	Username  string `json:"username"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	LGA       string `json:"LGA" gorm:"foreignkey:LGA(id)"`
	RoleName      string             `json:"role_name"`
}
type UserImage struct {
    ID           uint `gorm:"primaryKey"`
    UserID       uint
    ThumbNailURL string
    CreatedAt    time.Time
}

type EditProfileResponse struct {
	ID          uint   `json:"id"`
	Fullname    string `json:"fullname"`
	Username    string `json:"username"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequestMacAddress struct {
	Model
	MacAddress string `json:"mac_address" binding:"required"`
}
type ForgotPassword struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type UserIdResponse struct {
	ID uint `json:"id"`
}

type ResetPassword struct {
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}
type GoogleAuthResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type LoginResponse struct {
	UserResponse
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (u *User) VerifyPassword(password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
	if err != nil {
		return err // Passwords do not match
	}
	return nil // Passwords match
}

