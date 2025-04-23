package main

import (
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/iamlongalong/listenmail/handler"
	"github.com/iamlongalong/listenmail/pkg/dispatcher"
	"github.com/iamlongalong/listenmail/pkg/handlers"
	"github.com/iamlongalong/listenmail/pkg/server"
	"github.com/iamlongalong/listenmail/pkg/sources"
	"github.com/iamlongalong/listenmail/pkg/types"
	"gopkg.in/yaml.v3"
)

func main() {
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		log.Printf("config.yaml not found, writing default.")
		err = os.WriteFile("config.yaml", []byte(defaultConfig), os.ModePerm)
		if err != nil {
			log.Fatalf("Error write default config: %s", err)
			return
		}
	}
	// Read configuration
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
		return
	}

	var config types.ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Error parsing config: %v", err)
	}

	// Create dispatcher
	disp := dispatcher.New()
	defer disp.Close() // 确保在程序退出时关闭dispatcher

	// Add example handler
	if err := disp.AddHandlers(
		handlers.NewLogHandler(),
		handler.SaveHandler(config.Save.Dir),
		handler.CursorCodeHandler(),
	); err != nil {
		log.Fatalf("Error adding handler: %v", err)
	}

	// Create and start sources
	var activeSources []types.Source

	var src types.Source

	for _, cfg := range config.Sources.SMTP {
		if !cfg.Enabled {
			continue
		}

		src, err = sources.NewSMTPSource(cfg, disp)
		if err != nil {
			log.Printf("Error creating source %s: %v", cfg.Name, err)
		}
		if err := src.Start(); err != nil {
			log.Printf("Error starting source %s: %v", cfg.Name, err)
		}
		activeSources = append(activeSources, src)
	}

	for _, cfg := range config.Sources.IMAP {
		if !cfg.Enabled {
			continue
		}

		src, err = sources.NewIMAPSource(cfg, disp)
		if err != nil {
			log.Printf("Error creating source %s: %v", cfg.Name, err)
		}
		if err := src.Start(); err != nil {
			log.Printf("Error starting source %s: %v", cfg.Name, err)
		}
		activeSources = append(activeSources, src)
	}

	for _, cfg := range config.Sources.POP3 {
		if !cfg.Enabled {
			continue
		}

		src, err = sources.NewPOP3Source(cfg, disp)
		if err != nil {
			log.Printf("Error creating source %s: %v", cfg.Name, err)
		}
		if err := src.Start(); err != nil {
			log.Printf("Error starting source %s: %v", cfg.Name, err)
		}
		activeSources = append(activeSources, src)
	}

	for _, cfg := range config.Sources.MailHog {
		if !cfg.Enabled {
			continue
		}

		src, err = sources.NewMailHogSource(cfg, disp)
		if err != nil {
			log.Printf("Error creating source %s: %v", cfg.Name, err)
		}
		if err := src.Start(); err != nil {
			log.Printf("Error starting source %s: %v", cfg.Name, err)
		}
		activeSources = append(activeSources, src)
	}

	activeSources = append(activeSources, src)

	if len(activeSources) == 0 {
		log.Fatal("No sources were started")
	}

	s, err := server.New(server.Config{
		DBPath:        path.Join(config.Save.Dir, "emails.db"),
		AttachmentDir: path.Join(config.Save.Dir, "attachments"),
		Username:      config.Server.Username,
		Password:      config.Server.Password,
	})
	if err != nil {
		log.Fatalf("create server fail: %s", err)
		return
	}

	go func() {
		err = s.Run(config.Server.Addr)
		if err != nil {
			log.Fatalf("run server fail: %s", err)
		}
	}()

	log.Println("listener is running...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	if err := s.Close(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	// Stop all sources
	for _, src := range activeSources {
		if err := src.Stop(); err != nil {
			log.Printf("Error stopping source %s: %v", src.Name(), err)
		}
	}
}

var defaultConfig = `server:
  addr: "0.0.0.0:80"
  username: "admin"
  password: "admin"
save:
  dir: "./data"
sources:
  smtp:
    - name: local_smtp
      enabled: true

      address: ":25"
      domain: "0.0.0.0"
      read_timeout: 10s
      write_timeout: 10s
      max_message_bytes: 10485760  # 10MB
      max_recipients: 50
      allow_insecure_auth: true
`
