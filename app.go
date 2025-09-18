package main

import (
	"cid_retranslator/client"
	"cid_retranslator/config"
	"cid_retranslator/queue"
	"cid_retranslator/server"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"strings"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/natefinch/lumberjack.v2"
)

// App struct
type App struct {
	ctx         context.Context // Signal context for shutdown
	wailsCtx    context.Context // Wails context for runtime calls
	cfg         *config.Config
	appQueue    *queue.Queue
	tcpServer   *server.Server
	tcpClient   *client.Client
	logger      *slog.Logger
	fileLogger  *lumberjack.Logger // Store fileLogger for closing
	cancelfunc  context.CancelFunc
	wg          sync.WaitGroup
	logBuffer   []string
	logMu       sync.RWMutex
	startTime   time.Time
}

// NewApp creates a new App application struct
func NewApp() *App {
	cfg := config.New()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sharedQueue := queue.New(cfg.Queue.BufferSize)

	app := &App{
		ctx:        ctx,
		cfg:        cfg,
		appQueue:   queue.New(cfg.Queue.BufferSize),
		tcpServer:  server.New(&cfg.Server, sharedQueue, &cfg.CIDRules),
		tcpClient:  client.New(&cfg.Client, sharedQueue),
		cancelfunc: cancel,
		logBuffer:  make([]string, 0, 100),
		startTime:  time.Now(),
	}

	// Validate log file path and create directory if needed
	if cfg.Logging.Filename == "" {
		cfg.Logging.Filename = "cid_retranslator.log" // Default filename
	}
	logDir := filepath.Dir(cfg.Logging.Filename)
	if logDir != "." && logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory %s: %v\n", logDir, err)
		}
	}

	fileLogger := &lumberjack.Logger{
		Filename:   cfg.Logging.Filename,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
	}
	app.fileLogger = fileLogger // Store for closing later

	multiWriter := io.MultiWriter(os.Stdout, fileLogger)

	// Create custom handler for collecting full messages
	handler := &logHandler{
		app:     app,
		handler: slog.NewTextHandler(multiWriter, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}

	app.logger = slog.New(handler)
	slog.SetDefault(app.logger)

	// Log initialization to verify logging setup
	app.logger.Info("Logger initialized", "filename", cfg.Logging.Filename)

	return app
}

// logHandler and related methods
type logHandler struct {
	app     *App
	handler slog.Handler
}

func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *logHandler) Handle(ctx context.Context, r slog.Record) error {
	var msgBuilder strings.Builder
	msgBuilder.WriteString(r.Time.Format("2006-01-02 15:04:05.000"))
	msgBuilder.WriteString("\t")
	msgBuilder.WriteString(r.Level.String())
	msgBuilder.WriteString("\t")
	msgBuilder.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != slog.TimeKey && a.Key != slog.LevelKey && a.Key != slog.MessageKey {
			msgBuilder.WriteString("\t")
			msgBuilder.WriteString(a.Key)
			msgBuilder.WriteString("=")
			msgBuilder.WriteString(a.Value.String())
		}
		return true
	})

	h.app.logMu.Lock()
	h.app.logBuffer = append(h.app.logBuffer, msgBuilder.String())
	if len(h.app.logBuffer) > 100 {
		h.app.logBuffer = h.app.logBuffer[1:]
	}
	h.app.logMu.Unlock()

	err := h.handler.Handle(ctx, r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
	}
	return err
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &logHandler{app: h.app, handler: h.handler.WithAttrs(attrs)}
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	return &logHandler{app: h.app, handler: h.handler.WithGroup(name)}
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	// Save Wails context for runtime calls
	a.wailsCtx = ctx

	// Listen for window minimize events
	runtime.EventsOn(ctx, "window-state-changed", func(optionalData ...interface{}) {
		if len(optionalData) > 0 {
			if state, ok := optionalData[0].(string); ok && state == "minimised" {
				runtime.WindowHide(a.wailsCtx)
				a.logger.Info("Window minimized, hidden to system tray")
			}
		}
	})

	// Start system tray
	go func() {
		systray.Run(a.onReady, a.onExit)
	}()

	// Start TCP server and client
	a.wg.Add(2)
	go func() {
		defer a.wg.Done()
		a.tcpServer.Run(a.ctx)
	}()
	go func() {
		defer a.wg.Done()
		a.tcpClient.Run(a.ctx)
	}()
}

// onReady sets up the system tray menu
func (a *App) onReady() {
	systray.SetIcon(icon)
	systray.SetTitle("CID Retranslator")
	systray.SetTooltip("CID Retranslator Application")

	mShow := systray.AddMenuItem("Show", "Show the application window")
	mMinimize := systray.AddMenuItem("Minimize", "Minimize the application window")
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				a.ShowWindow()
			case <-mMinimize.ClickedCh:
				a.MinimizeWindow()
			case <-mQuit.ClickedCh:
				systray.Quit()
				a.Quit()
			}
		}
	}()
}

// onExit handles cleanup when the system tray exits
func (a *App) onExit() {
	a.logger.Info("System tray exited")
}

// ShowWindow shows the application window
func (a *App) ShowWindow() {
	if a.wailsCtx != nil {
		runtime.WindowShow(a.wailsCtx)
		a.logger.Info("Window shown from system tray")
	}
}

// MinimizeWindow hides the application window
func (a *App) MinimizeWindow() {
	if a.wailsCtx != nil {
		runtime.WindowHide(a.wailsCtx)
		a.logger.Info("Window minimized from system tray")
	}
}

// Quit terminates the application
func (a *App) Quit() {
	if a.wailsCtx != nil {
		runtime.Quit(a.wailsCtx)
		a.logger.Info("Application quit from system tray")
	}
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	a.logger.Info("Received shutdown signal, initiating graceful shutdown...")
	a.cancelfunc()
	a.tcpServer.Stop()
	a.tcpClient.Stop()
	a.wg.Wait()
	a.appQueue.Close()
	if a.fileLogger != nil {
		if err := a.fileLogger.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close file logger: %v\n", err)
		}
	}
	systray.Quit() // Ensure system tray is closed
	a.logger.Info("Program exited gracefully")
}

// Stats and related methods
type Stats struct {
	Accepted   int    `json:"accepted"`
	Rejected   int    `json:"rejected"`
	Uptime     string `json:"uptime"`
	Reconnects int    `json:"reconnects"`
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (a *App) GetStats() Stats {
	accepted, rejected, reconnects, _ := a.tcpClient.GetQueueStats()
	uptime := time.Since(a.startTime).Truncate(time.Second)
	return Stats{
		Accepted:   accepted,
		Rejected:   rejected,
		Uptime:     formatDuration(uptime),
		Reconnects: reconnects,
	}
}

func (a *App) GetLogs() []string {
	a.logMu.RLock()
	defer a.logMu.RUnlock()
	return append([]string{}, a.logBuffer...)
}

func (a *App) GetDevices() []server.Device {
	devices := a.tcpServer.GetDevices()
	return devices
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}