// handlers/admin.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminAPI регистрирует CRUD-роуты /admin/*
func AdminAPI(router *gin.RouterGroup, db *gorm.DB) {
	// список всех пользовательских таблиц
	router.GET("/tables", func(c *gin.Context) {
		var tables []string
		db.Raw(`
		    SELECT table_name
		    FROM information_schema.tables
		    WHERE table_schema = 'public'
		      AND table_type   = 'BASE TABLE'
		    ORDER BY table_name
		`).Scan(&tables)
		c.JSON(http.StatusOK, tables)
	})

	// схема колонок
	router.GET("/schema", func(c *gin.Context) {
		var meta []struct {
			Table  string `json:"table"`    // alias для table_name
			Column string `json:"column"`   // alias для column_name
			Null   string `json:"nullable"` // is_nullable
			Type   string `json:"type"`     // data_type
		}
		db.Raw(`
			SELECT
				table_name  AS table,
				column_name AS column,
				is_nullable AS nullable,
				data_type   AS type
			FROM information_schema.columns
			WHERE table_schema = 'public'
		`).Scan(&meta)
		c.JSON(http.StatusOK, meta)
	})

	// универсальный CRUD для любой таблицы
	rAny := func(c *gin.Context) {
		tbl := c.Param("table")
		sess := db.Session(&gorm.Session{PrepareStmt: false})

		switch c.Request.Method {
		case http.MethodGet:
			var rows []map[string]any
			if err := sess.Table(tbl).Find(&rows).Error; err != nil {
				c.AbortWithError(400, err)
				return
			}
			c.JSON(http.StatusOK, rows)

		case http.MethodPost:
			var body map[string]any
			if err := c.ShouldBindJSON(&body); err != nil {
				c.AbortWithError(400, err)
				return
			}
			if err := sess.Table(tbl).
				Omit("id").
				Create(body).Error; err != nil {
				c.AbortWithError(400, err)
				return
			}
			c.Status(http.StatusCreated)

		case http.MethodPut:
			// обновление — JSON-объект с полем id
			var body map[string]any
			if err := c.ShouldBindJSON(&body); err != nil {
				c.AbortWithError(400, err)
				return
			}
			// id обязательное
			idVal, ok := body["id"]
			if !ok {
				c.String(400, "field id is required for update")
				return
			}
			delete(body, "id")
			if err := sess.Table(tbl).
				Where("id = ?", idVal).
				Updates(body).Error; err != nil {
				c.AbortWithError(400, err)
				return
			}
			c.Status(http.StatusOK)

		case http.MethodDelete:
			// удаление по ?id=…
			id := c.Query("id")
			if id == "" {
				c.String(400, "id query required")
				return
			}
			if err := sess.Table(tbl).Delete(nil, "id = ?", id).Error; err != nil {
				c.AbortWithError(400, err)
				return
			}
			c.Status(http.StatusNoContent)
		}
	}

	// список внешних ключей для таблицы
	router.GET("/fkeys", func(c *gin.Context) {
		tbl := c.Query("table")
		var fk []struct {
			ChildCol  string `json:"child_col"`
			Parent    string `json:"parent"`
			ParentCol string `json:"parent_col"`
		}
		db.Raw(`
			SELECT
				kcu.column_name AS child_col,
				ccu.table_name  AS parent,
				ccu.column_name AS parent_col
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON tc.constraint_name = kcu.constraint_name
			JOIN information_schema.constraint_column_usage ccu
			  ON ccu.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY'
			  AND tc.table_name = ?
		`, tbl).Scan(&fk)
		c.JSON(http.StatusOK, fk)
	})

	// список id (для формы UPDATE)
	router.GET("/rowids", func(c *gin.Context) {
		tbl := c.Query("table")
		var ids []int64
		db.Table(tbl).Select("id").Order("id").Scan(&ids)
		c.JSON(http.StatusOK, ids)
	})

	// выполнение произвольного SQL
	router.POST("/sql", func(c *gin.Context) {
		var in struct{ SQL string `json:"sql"` }
		if c.ShouldBindJSON(&in) != nil || in.SQL == "" {
			c.String(http.StatusBadRequest, "sql required")
			return
		}
		if err := db.Exec(in.SQL).Error; err != nil {
			c.AbortWithError(400, err)
			return
		}
		c.Status(http.StatusOK)
	})

	// регистрация маршрутов
	router.GET("/:table", rAny)
	router.POST("/:table", rAny)
	router.PUT("/:table", rAny)
	router.DELETE("/:table", rAny)
}
