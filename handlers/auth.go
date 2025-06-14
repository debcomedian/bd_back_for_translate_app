package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"bd_back_for_translate_app/auth"
)

//======================================================================
//  /auth/login
//  — проверяет логин / пароль, заданные в переменных окружения
//    ADMIN_LOGIN  – имя пользователя (например, langadmin)
//    ADMIN_HASH   – bcrypt-хеш пароля
//======================================================================

func AuthEndpoints(r *gin.RouterGroup) {

	adminLogin := os.Getenv("ADMIN_LOGIN")
	adminHash := os.Getenv("ADMIN_HASH")

	if adminLogin == "" || adminHash == "" {
		log.Fatal("[AUTH] ADMIN_LOGIN / ADMIN_HASH must be set")
	}

	r.POST("/login", func(c *gin.Context) {

		var in struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			log.Printf("[AUTH] bad request body: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		log.Printf("[AUTH] login attempt: login=%q", in.Login)

		if in.Login != adminLogin {
			log.Printf("[AUTH] unauthorized login: %q", in.Login)
			c.Status(http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(adminHash), []byte(in.Password)); err != nil {
			log.Printf("[AUTH] invalid password for user=%q: %v", in.Login, err)
			c.Status(http.StatusUnauthorized)
			return
		}

		token, err := auth.Sign(1, "admin")
		if err != nil {
			log.Printf("[AUTH] token generation error: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		log.Printf("[AUTH] successful login: %q", in.Login)
		c.JSON(http.StatusOK, gin.H{"token": token})
	})
}
