package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Instance struct {
	Logger    *zerolog.Logger
	GinEngine *gin.Engine
}

var Inst Instance
