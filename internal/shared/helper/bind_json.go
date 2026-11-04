package helper

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// helper
func BindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Datos inválidos",
			"errors":  HumanValidationErrors(err),
		})
		return false
	}
	return true
}

// handler
// var req RegisterRequest
// if !helper.BindJSON(c, &req) {
// 	return
// }
