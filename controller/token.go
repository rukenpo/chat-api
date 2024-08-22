package controller

import (
	"fmt"
	"log"
	"net/http"
	"one-api/common"
	"one-api/common/config"
	"one-api/common/network"
	"one-api/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	p, _ := strconv.Atoi(c.Query("p"))
	size, _ := strconv.Atoi(c.Query("size"))
	if p < 0 {
		p = 0
	}
	if size <= 0 {
		size = config.ItemsPerPage
	} else if size > 100 {
		size = 100
	}
	tokens, err := model.GetAllUserTokens(userId, p*size, size)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tokens,
	})
	return
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")
	tokens, err := model.SearchUserTokens(userId, keyword, token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tokens,
	})
	return
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token,
	})
	return
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	var expiresAt int64
	if token.ExpiryMode == "first_use" {
		if token.FirstUsedTime > 0 {
			expiresAt = token.FirstUsedTime + token.Duration // 转换为秒
		} else {
			expiresAt = 0 // 尚未使用
		}
	} else {
		expiresAt = token.ExpiredTime
	}
	if expiresAt == -1 {
		expiresAt = 0 // 处理永不过期的情况
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiresAt * 1000,
	})
}

func AddToken(c *gin.Context) {
	token := model.Token{}
	userId := c.GetInt("id")
	err := c.ShouldBindJSON(&token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := validateToken(c, token); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if token.Group != "" {
		role := model.GetRole(userId)
		if role < 10 {
			if _, exists := common.GroupUserRatio[token.Group]; !exists {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "无效的用户组",
				})
				return
			}
		} else {
			if _, exists := common.GroupRatio[token.Group]; !exists {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "无效的用户组",
				})
				return
			}
		}
	}
	cleanToken := model.Token{
		UserId:         c.GetInt("id"),
		Name:           token.Name,
		Key:            common.GenerateKey(),
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    token.ExpiredTime,
		RemainQuota:    token.RemainQuota,
		UnlimitedQuota: token.UnlimitedQuota,
		Group:          token.Group,
		BillingEnabled: token.BillingEnabled,
		Models:         token.Models,
		FixedContent:   token.FixedContent,
		ExpiryMode:     token.ExpiryMode,
		Duration:       token.Duration,
	}
	if cleanToken.ExpiryMode == "first_use" {
		cleanToken.ExpiredTime = -1
	}
	err = cleanToken.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateToken(c *gin.Context) {
    userId := c.GetInt("id")
    statusOnly := c.Query("status_only")
    billingStrategyOnly := c.Query("billing_strategy_only")
    token := model.Token{}
    err := c.ShouldBindJSON(&token)
    if err != nil {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": "解析请求数据失败: " + err.Error(),
        })
        return
    }

    cleanToken, err := model.GetTokenByIds(token.Id, userId)
    if err != nil {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": "获取令牌失败: " + err.Error(),
        })
        return
    }

    if statusOnly != "" {
        // 只更新状态
        cleanToken.Status = token.Status
    } else if billingStrategyOnly != "" {
        // 只更新计费策略
        cleanToken.BillingEnabled = token.BillingEnabled
    } else {
        // 更新所有字段
        cleanToken.Name = token.Name
        cleanToken.ExpiredTime = token.ExpiredTime
        cleanToken.RemainQuota = token.RemainQuota
        cleanToken.UnlimitedQuota = token.UnlimitedQuota
        cleanToken.Group = token.Group
        cleanToken.Models = token.Models
        cleanToken.FixedContent = token.FixedContent
        cleanToken.Subnet = token.Subnet
        cleanToken.ExpiryMode = token.ExpiryMode
        cleanToken.Duration = token.Duration

        if cleanToken.ExpiryMode == "first_use" {
            cleanToken.ExpiredTime = -1
        }
    }

    // 验证更新后的令牌
    if err := validateToken(c, *cleanToken); err != nil {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": "验证令牌失败: " + err.Error(),
        })
        return
    }

    // 检查令牌状态
    if cleanToken.Status == common.TokenStatusEnabled {
        if cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
            c.JSON(http.StatusOK, gin.H{
                "success": false,
                "message": "令牌已过期，无法启用，请先修改令牌过期时间，或者设置为永不过期",
            })
            return
        }
        if cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
            c.JSON(http.StatusOK, gin.H{
                "success": false,
                "message": "令牌可用额度已用尽，无法启用，请先修改令牌剩余额度，或者设置为无限额度",
            })
            return
        }
    }

    err = cleanToken.Update()
    if err != nil {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": "更新令牌失败: " + err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "令牌更新成功",
        "data":    cleanToken,
    })
}


func UpdateTokenBillingStrategy(c *gin.Context) {
	userId := c.GetInt("id")
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid token ID",
			"data":    nil,
		})
		return
	}
	log.Println(userId, tokenId)

	// 使用Token结构体的部分实例来绑定billing_enabled字段
	var partialToken struct {
		BillingEnabled int `json:"billing_enabled"` // 前端传来的是1或0
	}
	err = c.ShouldBindJSON(&partialToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	// 获取token对象，确认它确实属于操作的用户
	cleanToken, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Token not found",
		})
		return
	}

	// 将整数值转换为布尔值
	billingEnabled := false
	if partialToken.BillingEnabled == 1 {
		billingEnabled = true
	}

	// 更新BillingEnabled字段
	cleanToken.BillingEnabled = billingEnabled
	err = cleanToken.UpdateTokenBilling()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update billing strategy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Billing strategy updated successfully",
		"data":    cleanToken,
	})
}
func validateToken(c *gin.Context, token model.Token) error {
	if len(token.Name) > 30 {
		return fmt.Errorf("令牌名称过长")
	}
	if token.Subnet != nil && *token.Subnet != "" {
		err := network.IsValidSubnets(*token.Subnet)
		if err != nil {
			return fmt.Errorf("无效的网段：%s", err.Error())
		}
	}
	if token.ExpiryMode != "fixed" && token.ExpiryMode != "first_use" {
		return fmt.Errorf("无效的过期模式")
	}
	if token.ExpiryMode == "first_use" {
		if token.Duration <= 0 {
			return fmt.Errorf("首次使用模式下，有效期必须大于0")
		}
		if token.Duration > 8760*3600 { // 最长一年
			return fmt.Errorf("有效期不能超过一年")
		}
	}
	return nil
}

func UseTokenFirstTime(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")

	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Token not found",
		})
		return
	}

	if token.ExpiryMode == "first_use" && token.FirstUsedTime == 0 {
		token.FirstUsedTime = common.GetTimestamp()
		err = token.Update()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to update token",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Token used for the first time",
			"data": gin.H{
				"first_used_time": token.FirstUsedTime,
				"expires_at":      (token.FirstUsedTime + token.Duration) * 1000,
			},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Token already used or not in first_use mode",
			"data": gin.H{
				"first_used_time": token.FirstUsedTime,
				"expires_at":      token.ExpiredTime * 1000,
			},
		})
	}
}
