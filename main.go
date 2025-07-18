package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/daodao97/xgo/utils"
	"github.com/daodao97/xgo/xapp"
	"github.com/daodao97/xgo/xdb"
	"github.com/daodao97/xgo/xlog"
	"github.com/daodao97/xgo/xrequest"

	"companions/internal/admin"
	"companions/internal/api"
	"companions/internal/conf"
	"companions/internal/dao"
	"companions/internal/wss"
)

var Version string

func init() {
	xrequest.SetRequestDebug(true)
	if !utils.IsGoRun() {
		xlog.SetLogger(xlog.StdoutJson(xlog.WithLevel(slog.LevelDebug)))
	}
}

func main() {
	app := xapp.NewApp().
		AddStartup(
			conf.InitConf,
			func() error {
				return xdb.Inits(conf.Get().Database)
			},
		).
		AfterStarted(func() {
			xlog.Debug("version", xlog.String("version", Version))
			dao.Init()
		}).
		AddServer(xapp.NewHttp(xapp.Args.Bind, h))

	if err := app.Run(); err != nil {
		fmt.Printf("Application error: %v\n", err)
	}
}

func h() http.Handler {
	e := xapp.NewGin()
	e.Static("/static", "assets/static")
	e.LoadHTMLGlob("assets/*.html")

	wss.SetupRouter(e)
	api.SetupRouter(e)
	admin.SetupRouter(e)
	return e.Handler()
}
