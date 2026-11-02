package web

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/bitxeno/atvloadly/internal/app"
	"github.com/bitxeno/atvloadly/internal/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func Run(addr string, port int) error {
	server := fiber.New(fiber.Config{
		BodyLimit: 1024 * 1024 * 1024,
	})
	setup(server)

	if err := server.Listen(fmt.Sprintf("%s:%d", addr, port)); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

// RunListener serves the web app on an existing listener, which the desktop
// app uses to bind a random loopback port before opening its window. It
// returns when the listener closes.
func RunListener(ln net.Listener) error {
	server := fiber.New(fiber.Config{
		BodyLimit: 1024 * 1024 * 1024,
	})
	setup(server)

	if err := server.Listener(ln); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func setup(server *fiber.App) {
	auth, err := loadManagementAuth(filepath.Join(app.Config.Server.DataDir, "keys"))
	if err != nil {
		panic(err)
	}
	auth.mount(server)
	server.Get("/source", func(c *fiber.Ctx) error { return c.SendFile("/opt/songlib-source.tar.gz") })
	server.Get("/license", func(c *fiber.Ctx) error {
		data, err := os.ReadFile("/opt/LICENSE")
		if err != nil {
			return c.SendStatus(503)
		}
		c.Type("text")
		return c.Send(data)
	})
	mountSongLib(server)

	// set fiber web server access log
	server.Use(logger.New(logger.Config{Format: "${time} ${method} ${path} ${status}\n"}))
	accessWriter := log.CreateRollingLogFile(app.Config.Log.AccessLog)
	if accessWriter != nil {
		server.Use(logger.New(logger.Config{
			Output: accessWriter,
			Format: "${time} ${method} ${path} ${status}\n",
		}))
		log.Infof("Web access log file path: %s", app.Config.Log.AccessLog)
	}

	// A panicking handler answers 500 instead of stopping the server. Recover
	// runs inside the access loggers so they record the 500.
	server.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e any) {
			log.Errorf("panic in %s %s: %v\n%s", c.Method(), c.Path(), e, debug.Stack())
		},
	}))

	route(server)
}
