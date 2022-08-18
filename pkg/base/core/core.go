package core

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leilei3167/basic/pkg/errors"
)

//提供通用的 响应结构

type ErrResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Reference string `json:"reference,omitempty"`
}

func WriteResponse(c *gin.Context, err error, data any) {
	if err != nil {
		//日志记录可以记录详细信息(调用堆栈)
		log.Printf("%#+v", err)
		//错误必须提前注册到errors中,返回至前端的是脱敏的信息
		coder := errors.ParseCoder(err)
		c.JSON(coder.HTTPStatus(), ErrResponse{
			Code:      coder.Code(),
			Message:   coder.String(),
			Reference: coder.Reference(),
		})
		return
	}
	c.JSON(http.StatusOK, data)
}
