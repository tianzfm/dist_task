package handler

import (
	"net/http"
	"strconv"
	"time"

	"dist_task/internal/engine"
	"dist_task/internal/model"
	"dist_task/internal/repository"
	"dist_task/internal/retry"
	"dist_task/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	flowRepo       *repository.FlowRepository
	instanceRepo   *repository.InstanceRepository
	taskRepo       *repository.TaskRepository
	exceptionRepo  *repository.ExceptionRepository
	logRepo        *repository.LogRepository
	engine         *engine.Engine
	retryScheduler *retry.RetryScheduler
}

func NewHandler(
	flowRepo *repository.FlowRepository,
	instanceRepo *repository.InstanceRepository,
	taskRepo *repository.TaskRepository,
	exceptionRepo *repository.ExceptionRepository,
	logRepo *repository.LogRepository,
	eng *engine.Engine,
	retryScheduler *retry.RetryScheduler,
) *Handler {
	return &Handler{
		flowRepo:       flowRepo,
		instanceRepo:   instanceRepo,
		taskRepo:       taskRepo,
		exceptionRepo:  exceptionRepo,
		logRepo:        logRepo,
		engine:         eng,
		retryScheduler: retryScheduler,
	}
}

type CreateFlowRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	FlowType    string `json:"flow_type" binding:"required"`
	Definition  string `json:"definition" binding:"required"`
	CreateUser  string `json:"create_user" binding:"required"`
}

func (h *Handler) CreateFlow(c *gin.Context) {
	var req CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	flow := &model.TaskGroupFlow{
		ID:          generateID(),
		Name:        req.Name,
		Description: req.Description,
		FlowType:    req.FlowType,
		Definition:  req.Definition,
		CreateUser:  req.CreateUser,
		UpdatedUser: req.CreateUser,
	}

	if err := h.flowRepo.Create(flow); err != nil {
		logger.Error().Err(err).Msg("create flow failed")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "create flow failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    flow,
	})
}

func (h *Handler) ListFlows(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	flows, total := h.flowRepo.List(offset, pageSize)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list": flows,
			"pagination": gin.H{
				"page":      page,
				"page_size": pageSize,
				"total":     total,
			},
		},
	})
}

func (h *Handler) GetFlow(c *gin.Context) {
	id := c.Param("id")

	flow, err := h.flowRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "flow not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    flow,
	})
}

type StartTransactionRequest struct {
	InstanceID string                 `json:"instance_id" binding:"required"`
	FlowID     string                 `json:"flow_id" binding:"required"`
	Params     map[string]interface{} `json:"params"`
}

func (h *Handler) StartTransaction(c *gin.Context) {
	var req StartTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 幂等检查
	existing, err := h.instanceRepo.GetByID(req.InstanceID)
	if err == nil && existing != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"instance_id": req.InstanceID,
				"status":      existing.Status,
			},
		})
		return
	}

	// 获取 flow 定义
	flow, err := h.flowRepo.GetByID(req.FlowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "flow not found"})
		return
	}

	// 创建 instance
	now := time.Now()
	instance := &model.TaskGroupInstance{
		ID:        req.InstanceID,
		FlowID:    req.FlowID,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.instanceRepo.Create(instance); err != nil {
		logger.Error().Err(err).Msg("create instance failed")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "create instance failed"})
		return
	}

	// 异步执行
	go func() {
		ctx := c.Request.Context()
		params := req.Params
		if params == nil {
			params = make(map[string]interface{})
		}
		if err := h.engine.Execute(ctx, instance, flow, params); err != nil {
			logger.Error().Err(err).Str("instance_id", req.InstanceID).Msg("execute failed")
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"instance_id": req.InstanceID,
			"flow_id":     req.FlowID,
			"status":      "pending",
			"created_at":  now,
		},
	})
}

func (h *Handler) GetTransaction(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "instance not found"})
		return
	}

	tasks, _ := h.taskRepo.ListByGroupID(id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"instance_id":  id,
			"flow_id":      instance.FlowID,
			"status":       instance.Status,
			"tasks":        tasks,
			"created_at":   instance.CreatedAt,
			"completed_at": instance.CompletedAt,
		},
	})
}

func (h *Handler) ListExceptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	var handled *bool
	if handledStr := c.Query("handled"); handledStr != "" {
		h := handledStr == "true"
		handled = &h
	}

	exceptions, total := h.exceptionRepo.List(offset, pageSize, handled)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list": exceptions,
			"pagination": gin.H{
				"page":      page,
				"page_size": pageSize,
				"total":     total,
			},
		},
	})
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"status": "ok"},
	})
}

func (h *Handler) RetryTransaction(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "instance not found"})
		return
	}

	if instance.Status != "failed" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "only failed transactions can be retried"})
		return
	}

	flow, err := h.flowRepo.GetByID(instance.FlowID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "flow not found"})
		return
	}

	instance.Status = "pending"
	h.instanceRepo.Update(instance)

	go func() {
		ctx := c.Request.Context()
		if err := h.engine.Execute(ctx, instance, flow, map[string]interface{}{}); err != nil {
			logger.Error().Err(err).Str("instance_id", id).Msg("retry transaction failed")
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"instance_id": id,
			"status":      "pending",
		},
	})
}

func (h *Handler) HandleException(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Remark string `json:"remark"`
	}
	c.ShouldBindJSON(&req)

	exception, err := h.exceptionRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "exception not found"})
		return
	}

	now := time.Now()
	exception.Handled = true
	exception.HandledAt = &now
	exception.HandledRemark = req.Remark
	h.exceptionRepo.Update(exception)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"exception_id": id,
			"handled":      true,
		},
	})
}

func (h *Handler) RetryException(c *gin.Context) {
	id := c.Param("id")

	exception, err := h.exceptionRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "exception not found"})
		return
	}

	if exception.Handled {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "handled exception cannot be retried"})
		return
	}

	if exception.RetryStrategy == "no_retry" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "this exception is configured to not retry"})
		return
	}

	now := time.Now()
	nextAt := now.Add(time.Duration(exception.RetryInterval) * time.Second)
	exception.RetryNextAt = &nextAt
	h.exceptionRepo.Update(exception)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"exception_id":    id,
			"retry_scheduled": true,
			"retry_next_at":   nextAt,
		},
	})
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
