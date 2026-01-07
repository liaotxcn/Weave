package features

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"weave/models"
	"weave/pkg"
	"weave/plugins/core"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Note 表示一条事件记录
type Note struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreatedTime time.Time `json:"created_time"`
	UpdatedTime time.Time `json:"updated_time"`
}

// NotePlugin 记事本插件
type NotePlugin struct {
	// 使用MySQL数据库存储
	mutex         sync.RWMutex // 读写锁用于并发控制
	pluginManager *core.PluginManager
}

// Name 返回插件名称
func (p *NotePlugin) Name() string {
	return "note"
}

// Description 返回插件描述
func (p *NotePlugin) Description() string {
	return "一个记事本插件，可以实现事件记录的增删查改功能"
}

// Version 返回插件版本
func (p *NotePlugin) Version() string {
	return "1.0.0"
}

// GetDependencies 返回依赖的插件
func (p *NotePlugin) GetDependencies() []string {
	return []string{} // 不依赖其他插件
}

// GetConflicts 返回冲突的插件
func (p *NotePlugin) GetConflicts() []string {
	return []string{} // 与其他插件无冲突
}

// SetPluginManager 设置插件管理器
func (p *NotePlugin) SetPluginManager(manager *core.PluginManager) {
	p.pluginManager = manager
}

// Init 初始化插件
func (p *NotePlugin) Init() error {
	// 插件初始化
	pkg.Debug("NotePlugin initialized", zap.String("plugin", p.Name()))
	return nil
}

// Shutdown 关闭插件
func (p *NotePlugin) Shutdown() error {
	// 插件关闭逻辑
	pkg.Debug("NotePlugin shutdown", zap.String("plugin", p.Name()))
	return nil
}

// OnEnable 插件启用时调用
func (p *NotePlugin) OnEnable() error {
	// 插件启用逻辑
	pkg.Debug("NotePlugin enabled", zap.String("plugin", p.Name()))
	return nil
}

// OnDisable 插件禁用时调用
func (p *NotePlugin) OnDisable() error {
	// 插件禁用逻辑
	pkg.Debug("NotePlugin disabled", zap.String("plugin", p.Name()))
	return nil
}

// RegisterRoutes 保留旧的方法以确保兼容性
// 在使用新的GetRoutes方法后，这个方法实际上不会被调用
func (p *NotePlugin) RegisterRoutes(router *gin.Engine) {
	// 这个方法在使用新的GetRoutes时不会被调用
	// 保留只是为了兼容性
	pkg.Debug("Old RegisterRoutes method called", zap.String("plugin", p.Name()), zap.String("message", "建议使用新的GetRoutes方法"))
}

// Execute 执行插件功能
func (p *NotePlugin) Execute(params map[string]interface{}) (interface{}, error) {
	action, ok := params["action"].(string)
	if !ok {
		action = "default"
	}

	// 从参数中提取用户ID
	userID := uint(1) // 默认用户ID
	if userIDParam, ok := params["user_id"].(string); ok {
		if id, err := strconv.ParseUint(userIDParam, 10, 32); err == nil {
			userID = uint(id)
		}
	}
	// 从参数中提取租户ID（可选）
	tenantID := uint(0)
	if tenantParam, ok := params["tenant_id"].(string); ok {
		if tid, err := strconv.ParseUint(tenantParam, 10, 32); err == nil {
			tenantID = uint(tid)
		}
	}

	switch action {
	case "list":
		page := 1
		pageSize := 10
		if pageParam, ok := params["page"].(float64); ok {
			page = int(pageParam)
		}
		if pageSizeParam, ok := params["page_size"].(float64); ok {
			pageSize = int(pageSizeParam)
		}
		return p.listNotes(userID, tenantID, page, pageSize)

	case "get":
		if noteID, ok := params["id"].(string); ok {
			return p.getNote(userID, tenantID, noteID)
		}
		return nil, errors.New("缺少笔记ID参数")

	case "create":
		if title, ok := params["title"].(string); ok && title != "" {
			if content, ok := params["content"].(string); ok && content != "" {
				return p.createNote(userID, tenantID, title, content)
			}
			return nil, errors.New("内容不能为空")
		}
		return nil, errors.New("标题不能为空")

	case "update":
		if noteID, ok := params["id"].(string); ok {
			if title, ok := params["title"].(string); ok && title != "" {
				if content, ok := params["content"].(string); ok && content != "" {
					return p.updateNote(userID, tenantID, noteID, title, content)
				}
				return nil, errors.New("内容不能为空")
			}
			return nil, errors.New("标题不能为空")
		}
		return nil, errors.New("缺少笔记ID参数")

	case "delete":
		if noteID, ok := params["id"].(string); ok {
			return p.deleteNoteHandler(userID, tenantID, noteID)
		}
		return nil, errors.New("缺少笔记ID参数")

	case "search":
		keyword := ""
		if keywordParam, ok := params["keyword"].(string); ok {
			keyword = keywordParam
		}
		page := 1
		pageSize := 10
		if pageParam, ok := params["page"].(float64); ok {
			page = int(pageParam)
		}
		if pageSizeParam, ok := params["page_size"].(float64); ok {
			pageSize = int(pageSizeParam)
		}
		return p.searchNotes(userID, tenantID, keyword, page, pageSize)

	default:
		return gin.H{
				"plugin":      p.Name(),
				"description": p.Description(),
				"version":     p.Version(),
				"available_actions": []string{
					"list - 列出笔记",
					"get - 获取单个笔记",
					"create - 创建新笔记",
					"update - 更新笔记",
					"delete - 删除笔记",
					"search - 搜索笔记",
				},
			},
			nil
	}
}

// listNotes 获取当前用户的所有笔记
func (p *NotePlugin) listNotes(userID uint, tenantID uint, page, pageSize int) (interface{}, error) {
	// 获取读锁
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var notes []models.Note
	var total int64

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	db := pkg.DB.Where("user_id = ? AND tenant_id = ?", userID, tenantID)

	if err := db.Model(&models.Note{}).Count(&total).Error; err != nil {
		pkg.Error("Database error when counting notes", zap.Error(err))
		return nil, fmt.Errorf("获取笔记列表失败，请稍后重试")
	}

	if err := db.Offset(offset).Limit(pageSize).Order("created_time DESC").Find(&notes).Error; err != nil {
		pkg.Error("Database error when fetching notes", zap.Error(err))
		return nil, fmt.Errorf("获取笔记列表失败，请稍后重试")
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return gin.H{
			"notes":      notes,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
		nil
}

// getNote 获取单个笔记
func (p *NotePlugin) getNote(userID uint, tenantID uint, noteID string) (interface{}, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 将string类型的noteID转换为uint类型
	id, err := strconv.ParseUint(noteID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("无效的笔记ID")
	}

	var note models.Note
	db := pkg.DB.Where("id = ? AND user_id = ? AND tenant_id = ?", uint(id), userID, tenantID)
	if err := db.First(&note).Error; err != nil {
		return nil, fmt.Errorf("笔记不存在或无权访问")
	}
	return note, nil
}

// createNote 创建新笔记
func (p *NotePlugin) createNote(userID uint, tenantID uint, title, content string) (interface{}, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	note := models.Note{
		Title:       title,
		Content:     content,
		UserID:      userID,
		TenantID:    tenantID,
		CreatedTime: time.Now(),
		UpdatedTime: time.Now(),
	}

	if err := pkg.DB.Create(&note).Error; err != nil {
		pkg.Error("Database error when creating note", zap.Error(err))
		return nil, fmt.Errorf("创建笔记失败，请稍后重试")
	}
	return note, nil
}

// updateNote 更新笔记
func (p *NotePlugin) updateNote(userID uint, tenantID uint, noteID, title, content string) (interface{}, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 将string类型的noteID转换为uint类型
	id, err := strconv.ParseUint(noteID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("无效的笔记ID")
	}

	var note models.Note
	db := pkg.DB.Where("id = ? AND user_id = ? AND tenant_id = ?", uint(id), userID, tenantID)
	if err := db.First(&note).Error; err != nil {
		return nil, fmt.Errorf("笔记不存在或无权访问")
	}

	note.Title = title
	note.Content = content
	note.UpdatedTime = time.Now()

	if err := pkg.DB.Save(&note).Error; err != nil {
		pkg.Error("Database error when updating note", zap.Error(err))
		return nil, fmt.Errorf("更新笔记失败，请稍后重试")
	}
	return note, nil
}

// deleteNoteHandler 删除笔记的处理器
func (p *NotePlugin) deleteNoteHandler(userID uint, tenantID uint, noteID string) (interface{}, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 将string类型的noteID转换为uint类型
	id, err := strconv.ParseUint(noteID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("无效的笔记ID")
	}

	var note models.Note
	db := pkg.DB.Where("id = ? AND user_id = ? AND tenant_id = ?", uint(id), userID, tenantID)
	if err := db.First(&note).Error; err != nil {
		return nil, fmt.Errorf("笔记不存在或无权访问")
	}

	if err := pkg.DB.Delete(&note).Error; err != nil {
		pkg.Error("Database error when deleting note", zap.Error(err))
		return nil, fmt.Errorf("删除笔记失败，请稍后重试")
	}

	return gin.H{"message": "删除成功"}, nil
}

// deleteNote 删除笔记（用户关联）
func (p *NotePlugin) deleteNote(userID uint, tenantID uint, noteID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 将string类型的noteID转换为uint类型
	id, err := strconv.ParseUint(noteID, 10, 32)
	if err != nil {
		return fmt.Errorf("无效的笔记ID")
	}

	var note models.Note
	db := pkg.DB
	if err := db.Where("id = ? AND user_id = ? AND tenant_id = ?", uint(id), userID, tenantID).First(&note).Error; err != nil {
		return errors.New("笔记不存在或无权访问")
	}

	if err := db.Delete(&note).Error; err != nil {
		return err
	}
	return nil
}

// searchNotes 搜索当前用户的笔记
func (p *NotePlugin) searchNotes(userID uint, tenantID uint, keyword string, page, pageSize int) (interface{}, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var notes []models.Note
	var total int64

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	query := "%" + keyword + "%"
	db := pkg.DB.Where("user_id = ? AND tenant_id = ? AND (title LIKE ? OR content LIKE ?)", userID, tenantID, query, query)

	if err := db.Model(&models.Note{}).Count(&total).Error; err != nil {
		pkg.Error("Database error when counting search results", zap.Error(err))
		return nil, fmt.Errorf("搜索笔记失败，请稍后重试")
	}

	if err := db.Offset(offset).Limit(pageSize).Order("created_time DESC").Find(&notes).Error; err != nil {
		pkg.Error("Database error when searching notes", zap.Error(err))
		return nil, fmt.Errorf("搜索笔记失败，请稍后重试")
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return gin.H{
			"notes":      notes,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
		nil
}

// 下面是核心业务方法的实现

// GetDefaultMiddlewares 返回插件的默认中间件
func (p *NotePlugin) GetDefaultMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}

// GetRoutes 返回插件的路由定义
func (p *NotePlugin) GetRoutes() []core.Route {
	return []core.Route{
		{
			Path:   "/",
			Method: "GET",
			Handler: func(c *gin.Context) {
				c.JSON(200, gin.H{
					"plugin":      p.Name(),
					"name":        p.Name(),
					"description": p.Description(),
					"version":     p.Version(),
					"endpoints": []string{
						"GET /plugins/note/ - 获取插件信息",
						"GET /plugins/note/notes - 获取所有笔记（需认证；按租户与用户隔离）",
						"GET /plugins/note/notes/:id - 获取单个笔记（需认证；按租户与用户隔离）",
						"POST /plugins/note/notes - 创建新笔记（需认证；按租户与用户隔离）",
						"PUT /plugins/note/notes/:id - 更新笔记（需认证；按租户与用户隔离）",
						"DELETE /plugins/note/notes/:id - 删除笔记（需认证；按租户与用户隔离）",
						"GET /plugins/note/notes/search - 搜索笔记（需认证；按租户与用户隔离）",
					},
				})
			},
			Description:  "获取插件信息",
			AuthRequired: false,
			Tags:         []string{"info", "metadata"},
		},
		{
			Path:   "/notes",
			Method: "GET",
			Handler: func(c *gin.Context) {
				userID := c.GetUint("user_id")
				tenantID := c.GetUint("tenant_id")
				page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
				pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

				result, err := p.listNotes(userID, tenantID, page, pageSize)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, result)
			},
			Description:  "获取所有笔记（支持分页和用户关联）",
			AuthRequired: true,
			Tags:         []string{"notes", "list"},
			Params: map[string]string{
				"page":      "页码，默认1",
				"page_size": "每页数量，默认10",
			},
		},
		{
			Path:   "/notes/:id",
			Method: "GET",
			Handler: func(c *gin.Context) {
				userID := c.GetUint("user_id")
				tenantID := c.GetUint("tenant_id")
				id := c.Param("id")

				result, err := p.getNote(userID, tenantID, id)
				if err != nil {
					c.JSON(404, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, result)
			},
			Description:  "获取单个笔记（用户关联）",
			AuthRequired: true,
			Tags:         []string{"notes", "get"},
			Params: map[string]string{
				"id": "笔记ID",
			},
		},
		{
			Path:   "/notes",
			Method: "POST",
			Handler: func(c *gin.Context) {
				userID := c.GetUint("user_id")
				tenantID := c.GetUint("tenant_id")

				var request struct {
					Title   string `json:"title" binding:"required,min=1,max=100"`
					Content string `json:"content" binding:"required,min=1"`
				}
				if err := c.ShouldBindJSON(&request); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				result, err := p.createNote(userID, tenantID, request.Title, request.Content)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				c.JSON(201, result)
			},
			Description:  "创建新笔记（用户关联）",
			AuthRequired: true,
			Tags:         []string{"notes", "create"},
		},
		{
			Path:   "/notes/:id",
			Method: "PUT",
			Handler: func(c *gin.Context) {
				userID := c.GetUint("user_id")
				tenantID := c.GetUint("tenant_id")
				id := c.Param("id")

				var request struct {
					Title   string `json:"title" binding:"required,min=1,max=100"`
					Content string `json:"content" binding:"required,min=1"`
				}
				if err := c.ShouldBindJSON(&request); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				result, err := p.updateNote(userID, tenantID, id, request.Title, request.Content)
				if err != nil {
					c.JSON(404, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, result)
			},
			Description:  "更新笔记（用户关联）",
			AuthRequired: true,
			Tags:         []string{"notes", "update"},
		},
		{
			Path:   "/notes/:id",
			Method: "DELETE",
			Handler: func(c *gin.Context) {
				userID := c.GetUint("user_id")
				tenantID := c.GetUint("tenant_id")
				id := c.Param("id")

				result, err := p.deleteNoteHandler(userID, tenantID, id)
				if err != nil {
					c.JSON(404, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, result)
			},
			Description:  "删除笔记（用户关联）",
			AuthRequired: true,
			Tags:         []string{"notes", "delete"},
		},
		{
			Path:   "/notes/search",
			Method: "GET",
			Handler: func(c *gin.Context) {
				userID := c.GetUint("user_id")
				tenantID := c.GetUint("tenant_id")
				keyword := c.DefaultQuery("keyword", "")
				page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
				pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

				result, err := p.searchNotes(userID, tenantID, keyword, page, pageSize)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, result)
			},
			Description:  "搜索笔记（支持分页和用户关联）",
			AuthRequired: true,
			Tags:         []string{"notes", "search"},
			Params: map[string]string{
				"keyword":   "搜索关键字",
				"page":      "页码，默认1",
				"page_size": "每页数量，默认10",
			},
		},
	}
}
