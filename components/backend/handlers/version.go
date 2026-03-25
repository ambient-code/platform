package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var version = "dev"

func SetVersion(v string) {
	version = v
}

func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": version})
}
