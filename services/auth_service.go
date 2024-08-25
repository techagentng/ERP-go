package services

import (
	"crypto/rand"
	// "encoding/json"
	"errors"
	"fmt"

	// "io/ioutil"
	_ "github.com/gin-gonic/gin"
	_ "github.com/golang-jwt/jwt"
	"github.com/techagentng/telair-erp/config"
	"github.com/techagentng/telair-erp/db"
	apiError "github.com/techagentng/telair-erp/errors"
	"github.com/techagentng/telair-erp/models"
	"github.com/techagentng/telair-erp/services/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"net/http"
)

//go:generate mockgen -destination=../mocks/auth_mock.go -package=mocks github.com/decagonhq/meddle-api/services AuthService

// AuthService interface
type AuthService interface {
	LoginUser(request *models.LoginRequest) (*models.LoginResponse, *apiError.Error)
	LoginMacAddressUser(loginRequest *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, *apiError.Error)
	SignupUser(request *models.User) (*models.User, error)
	// UpdateUserImageUrl(imagePath string) *apiError.Error
	GetUserProfile(userID uint) (*models.User, error)
	EditUserProfile(userID uint, userDetails *models.EditProfileResponse) error
	// FacebookSignInUser(token string) (*string, *apiError.Error)
	// VerifyEmail(token string) error
	SendEmailForPasswordReset(user *models.ForgotPassword) *apiError.Error
	ResetPassword(user *models.ResetPassword, token string) *apiError.Error
	GetAllUsers() ([]models.User, error)
	// DeleteUserByEmail(userEmail string) *apiError.Error
}

// authService struct
type authService struct {
	Config   *config.Config
	authRepo db.AuthRepository
}

// LoginMacAddressUser implements AuthService.
func (s *authService) LoginMacAddressUser(loginRequest *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, *apiError.Error) {
	panic("unimplemented")
}

// NewAuthService instantiate an authService
func NewAuthService(authRepo db.AuthRepository, conf *config.Config) AuthService {
	return &authService{
		Config:   conf,
		authRepo: authRepo,
	}
}

func (s *authService) SignupUser(user *models.User) (*models.User, error) {
	if user == nil {
		log.Println("SignupUser error: user is nil")
		return nil, errors.New("user is nil")
	}

	if user.Email == "" {
		log.Println("SignupUser error: email is empty")
		return nil, errors.New("email is empty")
	}

	// Check if the email already exists
	err := s.authRepo.IsEmailExist(user.Email)
	if err != nil {
		log.Printf("SignupUser error: %v", err)
		return nil, apiError.GetUniqueContraintError(err)
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("SignupUser error hashing password: %v", err)
		return nil, apiError.ErrInternalServerError
	}
	user.HashedPassword = string(hashedPassword)
	user.Password = "" // Clear the plain password

	// Create the user in the database
	user, err = s.authRepo.CreateUser(user)
	if err != nil {
		log.Printf("SignupUser error creating user: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	// Fetch the created user
	createdUser, err := s.authRepo.FindUserByEmail(user.Email)
	if err != nil {
		log.Printf("SignupUser error fetching created user: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	return createdUser, nil
}

func GenerateHashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

// LoginUser logs in a user and returns the login response
func (a *authService) LoginUser(loginRequest *models.LoginRequest) (*models.LoginResponse, *apiError.Error) {
	foundUser, err := a.authRepo.FindUserByEmail(loginRequest.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apiError.New("invalid email or password", http.StatusUnprocessableEntity)
		} else {
			log.Printf("error from database: %v", err)

			return nil, apiError.New("unable to find user", http.StatusInternalServerError)
		}
	}


	accessToken, refreshToken, err := jwt.GenerateTokenPair(foundUser.Email, a.Config.JWTSecret, foundUser.Role.Name, foundUser.ID)
	if err != nil {
		log.Printf("error generating token pair: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	return &models.LoginResponse{
		UserResponse: models.UserResponse{
			ID:        foundUser.ID,
			Email:     foundUser.Email,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authService) VerifyEmail(token string) error {
	claims, err := jwt.ValidateAndGetClaims(token, a.Config.JWTSecret)
	if err != nil {
		return apiError.New("invalid link", http.StatusUnauthorized)
	}
	email := claims["email"].(string)
	err = a.authRepo.VerifyEmail(email, token)
	return err
}

// func (a *authService) GetUserByID(id string) (*models.User, error) {
//     user, err := a.authRepo.FindByID(id)
//     if err != nil {
//         return nil, err
//     }
//     return user, nil
// }

// func (a *authService) UpdateUserImageUrl(imagePath string) *apiError.Error {
// 	// Update user's profile with the image URL
// 	var user models.User
// 	user.ThumbNailURL = imagePath

// 	err := a.authRepo.UpdateUserImage(&user)
// 	if err != nil {
// 		log.Printf("Error updating user image in database: %v", err)
// 		return &apiError.Error{
// 			Message: "Failed to update user profilxxe",
// 			Status:  0,
// 		}
// 	}
// 	return nil
// }

func GenerateRandomString() (string, error) {
	n := 5
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := fmt.Sprintf("%X", b)
	return s, nil
}

func (a *authService) GetUserProfile(userID uint) (*models.User, error) {
	// Call repository method to fetch user profile
	user, err := a.authRepo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *authService) EditUserProfile(userID uint, userDetail *models.EditProfileResponse) error {
	// Implement your business logic here, if needed
	// For example, you might want to perform validation on the user details before updating

	// Call the repository method to update user profile
	return a.authRepo.EditUserProfile(userID, userDetail)
}

func (a *authService) SendEmailForPasswordReset(user *models.ForgotPassword) *apiError.Error {
	return apiError.ErrBadRequest
}

func (a *authService) ResetPassword(user *models.ResetPassword, token string) *apiError.Error {
	return apiError.ErrBadRequest
}

func (s *authService) GetAllUsers() ([]models.User, error) {
	users, err := s.authRepo.GetAllUsers()
	if err != nil {
		return nil, fmt.Errorf("error getting all users: %w", err)
	}
	return users, nil
}
