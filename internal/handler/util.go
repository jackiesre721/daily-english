package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func getIntParam(c *gin.Context, key string, defaultVal int) int {
	val, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return defaultVal
	}
	return val
}
