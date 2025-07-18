package admin

import (
	"embed"
	"companions/internal/admin/hook"

	"github.com/daodao97/xgo/xadmin"
	"github.com/gin-gonic/gin"
)

//go:embed route.jsonc
var routes string

//go:embed schema
var schema embed.FS

func SetupRouter(e *gin.Engine) {
	xadmin.SetRoutes(routes)
	xadmin.InitSchema(schema)
	xadmin.SetAdminPath("/_")
	xadmin.SetJwt(&xadmin.JwtConf{
		Secret:      "xtpl_admin_jwt_secret",
		TokenExpire: 3600,
	})
	xadmin.SetWebSite(map[string]any{
		"title": "xtpl",
	})
	hook.RegHook()

	_ = xadmin.GinRoute(e)

	// 自定义路由
}
