package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wtitdn/renew_video/internal/controller/apierror"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/usecase"
	"github.com/wtitdn/renew_video/pkg/jwt"
)

type SocialHandler struct {
	service *usecase.SocialService
}

func NewSocialHandler(service *usecase.SocialService) *SocialHandler {
	return &SocialHandler{service: service}
}

func (h *SocialHandler) Follow(c *gin.Context) {
	var req entity.FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VloggerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	FollowerID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	social := &entity.Social{
		FollowerID: FollowerID,
		VloggerID:  req.VloggerID,
	}
	if err := h.service.Follow(c.Request.Context(), social); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}

func (h *SocialHandler) Unfollow(c *gin.Context) {
	var req entity.UnfollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VloggerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	FollowerID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	social := &entity.Social{
		FollowerID: FollowerID,
		VloggerID:  req.VloggerID,
	}
	if err := h.service.Unfollow(c.Request.Context(), social); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

func (h *SocialHandler) GetAllFollowers(c *gin.Context) {
	var req entity.GetAllFollowersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	vloggerID := req.VloggerID
	if vloggerID == 0 {
		accountID, err := jwt.GetAccountID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		vloggerID = accountID
	}

	followers, err := h.service.GetAllFollowers(c.Request.Context(), vloggerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if followers == nil {
		followers = []*entity.Account{}
	}
	followerCount, _ := h.service.CountFollowers(c.Request.Context(), vloggerID)
	c.JSON(http.StatusOK, entity.GetAllFollowersResponse{Followers: followers, FollowerCount: followerCount})
}

func (h *SocialHandler) GetAllVloggers(c *gin.Context) {
	var req entity.GetAllVloggersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	followerID := req.FollowerID
	if followerID == 0 {
		accountID, err := jwt.GetAccountID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		followerID = accountID
	}

	vloggers, err := h.service.GetAllVloggers(c.Request.Context(), followerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if vloggers == nil {
		vloggers = []*entity.Account{}
	}
	vloggerCount, _ := h.service.CountVloggers(c.Request.Context(), followerID)
	c.JSON(http.StatusOK, entity.GetAllVloggersResponse{Vloggers: vloggers, VloggerCount: vloggerCount})
}

func (h *SocialHandler) GetCounts(c *gin.Context) {
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	followerCount, _ := h.service.CountFollowers(c.Request.Context(), accountID)
	vloggerCount, _ := h.service.CountVloggers(c.Request.Context(), accountID)
	c.JSON(http.StatusOK, entity.SocialCounts{FollowerCount: followerCount, VloggerCount: vloggerCount})
}
