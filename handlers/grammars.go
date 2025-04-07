package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ************** Обработчики для грамматик **************

func GetGrammarHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT id, 
			       title_ru, title_en, title_de, 
			       description_ru, description_en, description_de, 
			       language 
			FROM ` + table
		rows, err := Dbpool.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		grammars := []Grammar{}
		for rows.Next() {
			var g Grammar
			if err := rows.Scan(
				&g.ID,
				&g.TitleRu, &g.TitleEn, &g.TitleDe,
				&g.DescriptionRu, &g.DescriptionEn, &g.DescriptionDe,
				&g.Language,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			grammars = append(grammars, g)
		}
		c.JSON(http.StatusOK, grammars)
	}
}

func CreateGrammarHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newGrammar Grammar
		if err := c.ShouldBindJSON(&newGrammar); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query := `
			INSERT INTO ` + table + ` (
				title_ru, title_en, title_de, 
				description_ru, description_en, description_de, 
				language
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`
		err := Dbpool.QueryRow(
			context.Background(),
			query,
			newGrammar.TitleRu, newGrammar.TitleEn, newGrammar.TitleDe,
			newGrammar.DescriptionRu, newGrammar.DescriptionEn, newGrammar.DescriptionDe,
			newGrammar.Language,
		).Scan(&newGrammar.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newGrammar)
	}
}

func UpdateGrammarHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		var updatedGrammar Grammar
		if err := c.ShouldBindJSON(&updatedGrammar); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedGrammar.ID = id
		query := `
			UPDATE ` + table + ` SET 
				title_ru=$1, title_en=$2, title_de=$3, 
				description_ru=$4, description_en=$5, description_de=$6, 
				language=$7
			WHERE id=$8
		`
		cmdTag, err := Dbpool.Exec(
			context.Background(),
			query,
			updatedGrammar.TitleRu, updatedGrammar.TitleEn, updatedGrammar.TitleDe,
			updatedGrammar.DescriptionRu, updatedGrammar.DescriptionEn, updatedGrammar.DescriptionDe,
			updatedGrammar.Language,
			updatedGrammar.ID,
		)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
		c.JSON(http.StatusOK, updatedGrammar)
	}
}

func DeleteGrammarHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		cmdTag, err := Dbpool.Exec(context.Background(), "DELETE FROM "+table+" WHERE id=$1", id)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ************** Обработчики для правил грамматики **************

func GetGrammarRulesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT id, grammar_id, 
			       rule_name_ru, rule_name_en, rule_name_de, 
			       rule_description_ru, rule_description_en, rule_description_de 
			FROM ` + table
		rows, err := Dbpool.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		rules := []GrammarRules{}
		for rows.Next() {
			var r GrammarRules
			if err := rows.Scan(
				&r.ID, &r.GrammarID,
				&r.RuleNameRu, &r.RuleNameEn, &r.RuleNameDe,
				&r.RuleDescriptionRu, &r.RuleDescriptionEn, &r.RuleDescriptionDe,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			rules = append(rules, r)
		}
		c.JSON(http.StatusOK, rules)
	}
}

func CreateGrammarRulesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newRule GrammarRules
		if err := c.ShouldBindJSON(&newRule); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query := `
			INSERT INTO ` + table + ` (
				grammar_id, 
				rule_name_ru, rule_name_en, rule_name_de, 
				rule_description_ru, rule_description_en, rule_description_de
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`
		err := Dbpool.QueryRow(
			context.Background(),
			query,
			newRule.GrammarID,
			newRule.RuleNameRu, newRule.RuleNameEn, newRule.RuleNameDe,
			newRule.RuleDescriptionRu, newRule.RuleDescriptionEn, newRule.RuleDescriptionDe,
		).Scan(&newRule.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newRule)
	}
}

func UpdateGrammarRulesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		var updatedRule GrammarRules
		if err := c.ShouldBindJSON(&updatedRule); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedRule.ID = id
		query := `
			UPDATE ` + table + ` SET 
				grammar_id=$1, 
				rule_name_ru=$2, rule_name_en=$3, rule_name_de=$4, 
				rule_description_ru=$5, rule_description_en=$6, rule_description_de=$7
			WHERE id=$8
		`
		cmdTag, err := Dbpool.Exec(
			context.Background(),
			query,
			updatedRule.GrammarID,
			updatedRule.RuleNameRu, updatedRule.RuleNameEn, updatedRule.RuleNameDe,
			updatedRule.RuleDescriptionRu, updatedRule.RuleDescriptionEn, updatedRule.RuleDescriptionDe,
			updatedRule.ID,
		)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
		c.JSON(http.StatusOK, updatedRule)
	}
}

func DeleteGrammarRulesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		cmdTag, err := Dbpool.Exec(context.Background(), "DELETE FROM "+table+" WHERE id=$1", id)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ************** Обработчики для примеров правил **************

func GetGrammarExamplesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT id, rule_id, 
			       example_ru, example_en, example_de 
			FROM ` + table
		rows, err := Dbpool.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		examples := []GrammarExamples{}
		for rows.Next() {
			var ex GrammarExamples
			if err := rows.Scan(
				&ex.ID, &ex.RuleID,
				&ex.ExampleRu, &ex.ExampleEn, &ex.ExampleDe,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			examples = append(examples, ex)
		}
		c.JSON(http.StatusOK, examples)
	}
}

func CreateGrammarExamplesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newExample GrammarExamples
		if err := c.ShouldBindJSON(&newExample); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query := `
			INSERT INTO ` + table + ` (
				rule_id, example_ru, example_en, example_de
			) VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err := Dbpool.QueryRow(
			context.Background(),
			query,
			newExample.RuleID,
			newExample.ExampleRu, newExample.ExampleEn, newExample.ExampleDe,
		).Scan(&newExample.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newExample)
	}
}

func UpdateGrammarExamplesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		var updatedExample GrammarExamples
		if err := c.ShouldBindJSON(&updatedExample); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedExample.ID = id
		query := `
			UPDATE ` + table + ` SET 
				rule_id=$1, 
				example_ru=$2, example_en=$3, example_de=$4
			WHERE id=$5
		`
		cmdTag, err := Dbpool.Exec(
			context.Background(),
			query,
			updatedExample.RuleID,
			updatedExample.ExampleRu, updatedExample.ExampleEn, updatedExample.ExampleDe,
			updatedExample.ID,
		)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
		c.JSON(http.StatusOK, updatedExample)
	}
}

func DeleteGrammarExamplesHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		cmdTag, err := Dbpool.Exec(context.Background(), "DELETE FROM "+table+" WHERE id=$1", id)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ************** Обработчики для исключений правил **************

func GetGrammarExceptionsHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT id, rule_id, 
			       description_ru, description_en, description_de, 
			       explanation_ru, explanation_en, explanation_de 
			FROM ` + table
		rows, err := Dbpool.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		exceptions := []GrammarExceptions{}
		for rows.Next() {
			var ex GrammarExceptions
			if err := rows.Scan(
				&ex.ID, &ex.RuleID,
				&ex.DescriptionRu, &ex.DescriptionEn, &ex.DescriptionDe,
				&ex.ExplanationRu, &ex.ExplanationEn, &ex.ExplanationDe,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			exceptions = append(exceptions, ex)
		}
		c.JSON(http.StatusOK, exceptions)
	}
}

func CreateGrammarExceptionsHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newException GrammarExceptions
		if err := c.ShouldBindJSON(&newException); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query := `
			INSERT INTO ` + table + ` (
				rule_id, 
				description_ru, description_en, description_de, 
				explanation_ru, explanation_en, explanation_de
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`
		err := Dbpool.QueryRow(
			context.Background(),
			query,
			newException.RuleID,
			newException.DescriptionRu, newException.DescriptionEn, newException.DescriptionDe,
			newException.ExplanationRu, newException.ExplanationEn, newException.ExplanationDe,
		).Scan(&newException.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newException)
	}
}

func UpdateGrammarExceptionsHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		var updatedException GrammarExceptions
		if err := c.ShouldBindJSON(&updatedException); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedException.ID = id
		query := `
			UPDATE ` + table + ` SET 
				rule_id=$1, 
				description_ru=$2, description_en=$3, description_de=$4, 
				explanation_ru=$5, explanation_en=$6, explanation_de=$7
			WHERE id=$8
		`
		cmdTag, err := Dbpool.Exec(
			context.Background(),
			query,
			updatedException.RuleID,
			updatedException.DescriptionRu, updatedException.DescriptionEn, updatedException.DescriptionDe,
			updatedException.ExplanationRu, updatedException.ExplanationEn, updatedException.ExplanationDe,
			updatedException.ID,
		)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
		c.JSON(http.StatusOK, updatedException)
	}
}

func DeleteGrammarExceptionsHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		cmdTag, err := Dbpool.Exec(context.Background(), "DELETE FROM "+table+" WHERE id=$1", id)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
