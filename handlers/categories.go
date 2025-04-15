package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetCategoriesHandler(c *gin.Context) {
	query := `
		SELECT id, name_en, name_ru, name_de, type, entity
		FROM categories
	`
	rows, err := Dbpool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var cat Category
		if err := rows.Scan(
			&cat.ID,
			&cat.NameEn, &cat.NameRu, &cat.NameDe,
			&cat.Type, &cat.Entity,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		categories = append(categories, cat)
	}
	c.JSON(http.StatusOK, categories)
}

func CreateCategoryHandler(c *gin.Context) {
	var newCat Category
	if err := c.ShouldBindJSON(&newCat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := `
		INSERT INTO categories (name_en, name_ru, name_de, type, entity)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := Dbpool.QueryRow(context.Background(), query,
		newCat.NameEn, newCat.NameRu, newCat.NameDe, newCat.Type, newCat.Entity,
	).Scan(&newCat.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newCat)
}

func UpdateCategoryHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var updatedCat Category
	if err := c.ShouldBindJSON(&updatedCat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updatedCat.ID = id
	query := `
		UPDATE categories
		SET name_en = $1, name_ru = $2, name_de = $3, type = $4, entity = $5
		WHERE id = $6
	`
	cmdTag, err := Dbpool.Exec(context.Background(), query,
		updatedCat.NameEn, updatedCat.NameRu, updatedCat.NameDe,
		updatedCat.Type, updatedCat.Entity, updatedCat.ID,
	)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, updatedCat)
}

func DeleteCategoryHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	cmdTag, err := Dbpool.Exec(context.Background(), "DELETE FROM categories WHERE id=$1", id)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}
