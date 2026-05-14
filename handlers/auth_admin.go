package handlers

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"

	"bd_back_for_translate_app/auth"
	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type adminLoginRequest struct {
	Login    string `json:"login"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

type adminLoginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	Login string `json:"login"`
}

func AdminLogin(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Некорректное тело JSON-запроса",
		})
		return
	}

	login := strings.TrimSpace(firstNonEmpty(req.Login, req.Username, req.Email))
	if login == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Логин не указан",
		})
		return
	}

	if DB != nil {
		adminUser, err := findActiveAdminUser(login)
		switch {
		case err == nil:
			if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(req.Password)); err == nil {
				token, signErr := auth.Sign(int(adminUser.ID), "admin")
				if signErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error":   "internal_error",
						"message": "Не удалось сформировать токен администратора",
					})
					return
				}
				c.JSON(http.StatusOK, adminLoginResponse{
					Token: token,
					Role:  "admin",
					Login: adminUser.Username,
				})
				return
			}
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Внутренняя ошибка авторизации администратора",
			})
			return
		}
	}

	if !checkEnvAdminCredentials(login, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Некорректные данные администратора",
		})
		return
	}

	token, err := auth.Sign(1, "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Внутренняя ошибка авторизации администратора",
		})
		return
	}

	c.JSON(http.StatusOK, adminLoginResponse{
		Token: token,
		Role:  "admin",
		Login: defaultAdminLogin(),
	})
}

func findActiveAdminUser(login string) (*models.AdminUser, error) {
	var adminUser models.AdminUser
	err := DB.Where("is_active = TRUE").Where("username = ? OR email = ?", login, login).First(&adminUser).Error
	if err != nil {
		return nil, err
	}
	return &adminUser, nil
}

func checkEnvAdminCredentials(login, password string) bool {
	expectedLogin := defaultAdminLogin()
	if subtle.ConstantTimeCompare([]byte(login), []byte(expectedLogin)) != 1 {
		return false
	}

	if hash := strings.TrimSpace(os.Getenv("ADMIN_HASH")); hash != "" {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}

	expectedPassword := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if expectedPassword == "" {
		expectedPassword = "admin"
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1
}

func defaultAdminLogin() string {
	login := strings.TrimSpace(os.Getenv("ADMIN_LOGIN"))
	if login == "" {
		return "admin"
	}
	return login
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
