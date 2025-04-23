package server

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iamlongalong/listenmail/pkg/types"
)

//go:embed web/*
var webFS embed.FS

// Server represents the HTTP server
type Server struct {
	db            *gorm.DB
	router        *gin.Engine
	attachmentDir string
	auth          struct {
		username string
		password string
	}
}

// Config represents the server configuration
type Config struct {
	DBPath        string
	Username      string
	Password      string
	AttachmentDir string
}

// New creates a new server instance
func New(config Config) (*Server, error) {
	// Open database connection
	db, err := gorm.Open(sqlite.Open(config.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database error: %v", err)
	}

	// Auto migrate schemas
	if err := db.AutoMigrate(&types.DBMail{}, &types.DBAddress{}, &types.DBAttachment{}); err != nil {
		return nil, fmt.Errorf("auto migrate error: %v", err)
	}

	s := &Server{
		db:            db,
		router:        gin.Default(),
		attachmentDir: config.AttachmentDir,
	}
	s.auth.username = config.Username
	s.auth.password = config.Password

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// Close closes the server resources
func (s *Server) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// basicAuth middleware
func (s *Server) basicAuth() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{
		s.auth.username: s.auth.password,
	})
}

// setupRoutes configures all the routes
func (s *Server) setupRoutes() {
	// 设置 gin 使用 embed.FS 中的模板
	templ := template.Must(template.New("").ParseFS(webFS, "web/*.html"))
	s.router.SetHTMLTemplate(templ)

	// 提供静态文件服务
	webStatic, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(fmt.Sprintf("failed to sub web filesystem: %v", err))
	}
	s.router.StaticFS("/static", http.FS(webStatic))

	// HTML routes
	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	s.router.GET("/mail/:id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "mail.html", nil)
	})

	// API routes with basic auth
	api := s.router.Group("/api", s.basicAuth())
	{
		// Mail routes
		api.GET("/mails", s.listMails)
		api.GET("/mails/:id", s.getMail)
		api.PUT("/mails/:id", s.updateMail)
		api.DELETE("/mails/:id", s.deleteMail)

		// Attachment routes
		api.GET("/attachments/:id", s.downloadAttachment)
	}
}

// QueryParams represents the query parameters for listing mails
type QueryParams struct {
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
	MailID    uint   `form:"mail_id"`
	MessageID string `form:"message_id"`
	FromDate  string `form:"from_date"`
	ToDate    string `form:"to_date"`
	From      string `form:"from"`
	To        string `form:"to"`
}

// listMails handles GET /api/mails
func (s *Server) listMails(c *gin.Context) {
	var params QueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate pagination parameters
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	// Base query
	query := s.db.Model(&types.DBMail{}).
		Preload("From").
		Preload("To").
		Preload("Cc").
		Preload("Bcc").
		Preload("Attachments")

	// Add filters
	if params.MailID != 0 {
		query = query.Where("id = ?", params.MailID)
	}
	if params.MessageID != "" {
		query = query.Where("message_id = ?", params.MessageID)
	}
	if params.FromDate != "" {
		query = query.Where("date >= ?", params.FromDate)
	}
	if params.ToDate != "" {
		query = query.Where("date <= ?", params.ToDate)
	}
	if params.From != "" {
		query = query.Joins("JOIN db_addresses a_from ON a_from.mail_id = db_mails.id AND a_from.type = 'from'").
			Where("a_from.address LIKE ?", "%"+params.From+"%")
	}
	if params.To != "" {
		query = query.Joins("JOIN db_addresses a_to ON a_to.mail_id = db_mails.id AND a_to.type = 'to'").
			Where("a_to.address LIKE ?", "%"+params.To+"%")
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated records
	var mails []types.DBMail
	if err := query.Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		Order("date DESC").
		Find(&mails).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to API response
	apiMails := make([]*types.APIMail, len(mails))
	for i := range mails {
		apiMails[i] = mails[i].ToAPIMail()
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
		"data":      apiMails,
	})
}

// getMail handles GET /api/mails/:id
func (s *Server) getMail(c *gin.Context) {
	id := c.Param("id")

	var mail types.DBMail
	err := s.db.Preload(clause.Associations).First(&mail, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mail not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mail.ToAPIMail())
}

// UpdateMailRequest represents the request body for updating a mail
type UpdateMailRequest struct {
	Subject   *string `json:"subject"`
	Priority  *string `json:"priority"`
	Important *bool   `json:"important"`
}

// updateMail handles PUT /api/mails/:id
func (s *Server) updateMail(c *gin.Context) {
	id := c.Param("id")

	var req UpdateMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})

	if req.Subject != nil {
		updates["subject"] = *req.Subject
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Important != nil {
		if *req.Important {
			updates["importance"] = "high"
		} else {
			updates["importance"] = "normal"
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	result := s.db.Model(&types.DBMail{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mail not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mail updated successfully"})
}

// deleteMail handles DELETE /api/mails/:id
func (s *Server) deleteMail(c *gin.Context) {
	id := c.Param("id")

	// Get attachments before deletion
	var attachments []types.DBAttachment
	if err := s.db.Where("mail_id = ?", id).Find(&attachments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete mail (will cascade delete addresses and attachments)
	result := s.db.Delete(&types.DBMail{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mail not found"})
		return
	}

	// Delete attachment files
	for _, att := range attachments {
		fullPath := filepath.Join(s.attachmentDir, att.Path)
		if err := os.Remove(fullPath); err != nil {
			// Log error but continue
			fmt.Printf("Error deleting attachment file %s: %v\n", fullPath, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mail deleted successfully"})
}

// downloadAttachment handles GET /api/attachments/:id
func (s *Server) downloadAttachment(c *gin.Context) {
	id := c.Param("id")

	var attachment types.DBAttachment
	if err := s.db.First(&attachment, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	fullPath := filepath.Join(s.attachmentDir, attachment.Path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment file not found"})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", attachment.Filename))
	c.Header("Content-Type", attachment.ContentType)
	c.File(fullPath)
}
