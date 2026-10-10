package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"ecommerce/services/auth/internal/model"
	"ecommerce/services/auth/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(req model.RegisterRequest) (*model.AuthResponse, error)
	Login(req model.LoginRequest) (*model.AuthResponse, error)
	GetProfile(userID uint) (*model.User, error)
}

type authService struct {
	userRepo repository.UserRepository
	rdb      *redis.Client
}

func NewAuthService(userRepo repository.UserRepository, rdb *redis.Client) AuthService {
	return &authService{
		userRepo: userRepo,
		rdb:      rdb,
	}
}

func (s *authService) createSession(userID uint, email string, name string) (string, error) {
	sessionID := uuid.New().String()

	sessionData := map[string]interface{}{
		"user_id": userID,
		"email":   email,
		"name":    name,
	}

	body, err := json.Marshal(sessionData)
	if err != nil {
		return "", err
	}

	if s.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		key := fmt.Sprintf("session:%s", sessionID)
		err := s.rdb.Set(ctx, key, string(body), 24*time.Hour).Err()
		if err != nil {
			log.Printf("[Warning] Failed to save session to Redis: %v", err)
		} else {
			log.Printf("[Redis Session] Created session %s for user %d", sessionID, userID)
		}
	}

	return sessionID, nil
}

func (s *authService) Register(req model.RegisterRequest) (*model.AuthResponse, error) {
	existingUser, _ := s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}
	if err := s.userRepo.Create(&user); err != nil {
		return nil, err
	}

	token, _ := generateJWT(user.ID, user.Email)
	sessionID, _ := s.createSession(user.ID, user.Email, user.Name)

	return &model.AuthResponse{
		SessionID: sessionID,
		Token:     token,
		User:      user,
	}, nil
}

func (s *authService) Login(req model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, _ := generateJWT(user.ID, user.Email)
	sessionID, _ := s.createSession(user.ID, user.Email, user.Name)

	return &model.AuthResponse{
		SessionID: sessionID,
		Token:     token,
		User:      *user,
	}, nil
}

func generateJWT(userID uint, email string) (string, error) {
	secretKey := []byte(os.Getenv("JWT_SECRET"))

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func (s *authService) GetProfile(userID uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}
