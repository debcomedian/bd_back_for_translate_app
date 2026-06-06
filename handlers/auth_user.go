package handlers

import (
	"net/http"
	"strings"

	"bd_back_for_translate_app/auth"
	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type publicUserView struct {
	ID       uint64  `json:"id"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Level    *string `json:"level,omitempty"`
}

type authResponse struct {
	Token string         `json:"token"`
	User  publicUserView `json:"user"`
}

func Register(c *gin.Context) {
	var req registerRequest
	if !bindJSON(c, &req) {
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(c, http.StatusUnprocessableEntity, "validation_error", "Необходимо указать username, email и password")
		return
	}
	if !isValidEmailAddress(req.Email) {
		writeError(c, http.StatusUnprocessableEntity, "validation_error", "Введите корректный email")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "password_hash_error", "Не удалось сформировать хеш пароля")
		return
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	if handleDBErr(c, DB.Create(&user).Error) {
		return
	}

	token, err := auth.Sign(int(user.ID), "user")
	if err != nil {
		writeError(c, http.StatusInternalServerError, "token_error", "Не удалось сформировать токен авторизации")
		return
	}

	respondCreated(c, authResponse{
		Token: token,
		User:  publicUserView{ID: user.ID, Username: user.Username, Email: user.Email, Level: user.Level},
	})
}

func Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}

	login := strings.TrimSpace(req.Login)
	var user models.User
	err := DB.Where("username = ? OR email = ?", login, strings.ToLower(login)).First(&user).Error
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "Некорректные учётные данные")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "Некорректные учётные данные")
		return
	}

	token, err := auth.Sign(int(user.ID), "user")
	if err != nil {
		writeError(c, http.StatusInternalServerError, "token_error", "Не удалось сформировать токен авторизации")
		return
	}

	respondOK(c, authResponse{
		Token: token,
		User:  publicUserView{ID: user.ID, Username: user.Username, Email: user.Email, Level: user.Level},
	})
}
