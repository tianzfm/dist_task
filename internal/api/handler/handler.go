package handler

import (
	"net/http"
	"strconv"
	"time"

	"dist_task/internal/engine"
	"dist_task/internal/model"
	"dist_task/internal/repository"
	"dist_task/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	flowRepo      *repository.FlowRepository
	instanceRepo  *repository.InstanceRepository
	taskRepo      *repository.TaskRepository
	exceptionRepo *repository.ExceptionRepository
	logRepo       *repository.LogRepository
	engine        *engine.Engine
}

func NewHandler(
	flowRepo *repository.FlowRepository,
	instanceRepo *repository.InstanceRepository,
	taskRepo *repository.TaskRepository,
	exceptionRepo *repository.ExceptionRepository,
	logRepo *repository.LogRepository,
	eng *engine.Engine,
) *Handler {
	return &Handler{
		flowRepo:      flowRepo,
		instanceRepo:  instanceRepo,
		taskRepo:      taskRepo,
		exceptionRepo: exceptionRepo,
		logRepo:       logRepo,
		engine:        eng,
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
		if err := h.engine.Execute(ctx, instance, flow); err != nil {
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

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
