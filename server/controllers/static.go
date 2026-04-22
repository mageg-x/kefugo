package controllers

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/utils/logger"
	"kefu-server/web"
)

// StaticController 负责注册前端静态资源路由（管理后台 + SDK）。
type StaticController struct{}

// Register 注册静态资源相关路由。
func (sc *StaticController) Register(r *gin.Engine) {
	adminFS, _ := fs.Sub(web.StaticFS, "admin")
	sdkFS, _ := fs.Sub(web.StaticFS, "sdk")

	r.GET("/", func(c *gin.Context) {
		if fileExists(adminFS, "home.html") {
			serveFile(c, adminFS, "home.html")
			return
		}
		serveFile(c, adminFS, "index.html")
	})
	r.GET("/home.html", func(c *gin.Context) {
		if fileExists(adminFS, "home.html") {
			serveFile(c, adminFS, "home.html")
			return
		}
		c.Status(http.StatusNotFound)
	})
	r.GET("/i-want.html", func(c *gin.Context) {
		if fileExists(adminFS, "i-want.html") {
			serveFile(c, adminFS, "i-want.html")
			return
		}
		c.Status(http.StatusNotFound)
	})
	r.GET("/login", func(c *gin.Context) {
		serveFile(c, adminFS, "index.html")
	})
	r.GET("/home", func(c *gin.Context) {
		serveFile(c, adminFS, "index.html")
	})
	r.GET("/home/*filepath", func(c *gin.Context) {
		serveFile(c, adminFS, "index.html")
	})
	r.GET("/admin", func(c *gin.Context) {
		serveFile(c, adminFS, "index.html")
	})
	r.GET("/admin/*filepath", func(c *gin.Context) {
		p := strings.TrimPrefix(c.Param("filepath"), "/")
		if p == "" {
			p = "index.html"
		}
		if !fileExists(adminFS, p) {
			p = "index.html"
		}
		serveFile(c, adminFS, p)
	})
	r.GET("/assets/*filepath", func(c *gin.Context) {
		p := strings.TrimPrefix(c.Param("filepath"), "/")
		if p == "" {
			logger.Errorf("static assets path empty")
			c.Status(http.StatusNotFound)
			return
		}
		assetPath := path.Join("assets", p)
		if !fileExists(adminFS, assetPath) {
			logger.Errorf("static asset not found path=%s", assetPath)
			c.Status(http.StatusNotFound)
			return
		}
		serveFile(c, adminFS, assetPath)
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		if !fileExists(adminFS, "favicon.ico") {
			logger.Errorf("static favicon not found")
			c.Status(http.StatusNotFound)
			return
		}
		serveFile(c, adminFS, "favicon.ico")
	})

	r.GET("/sdk/*filepath", func(c *gin.Context) {
		p := strings.TrimPrefix(c.Param("filepath"), "/")
		if p == "" {
			p = "widget.min.js"
		}
		if !fileExists(sdkFS, p) {
			logger.Errorf("static sdk file not found path=%s", p)
			c.Status(http.StatusNotFound)
			return
		}
		serveFile(c, sdkFS, p)
	})
}

// fileExists 判断文件是否存在于嵌入文件系统中。
func fileExists(fsys fs.FS, filePath string) bool {
	filePath = path.Clean(strings.TrimPrefix(filePath, "/"))
	_, err := fs.Stat(fsys, filePath)
	return err == nil
}

// serveFile 从嵌入文件系统读取并返回静态文件。
func serveFile(c *gin.Context, fsys fs.FS, filePath string) {
	filePath = path.Clean(strings.TrimPrefix(filePath, "/"))
	httpFS := http.FS(fsys)
	file, err := httpFS.Open(filePath)
	if err != nil {
		logger.Errorf("static file open failed path=%s err=%v", filePath, err)
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		logger.Errorf("static file stat failed path=%s err=%v", filePath, err)
		c.Status(http.StatusNotFound)
		return
	}
	if filePath == "index.html" {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}
	http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), file)
}
