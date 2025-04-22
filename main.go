package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iamlongalong/listenmail/handler"
	"github.com/iamlongalong/listenmail/pkg/dispatcher"
	"github.com/iamlongalong/listenmail/pkg/handlers"
	"github.com/iamlongalong/listenmail/pkg/sources"
	"github.com/iamlongalong/listenmail/pkg/types"
	"gopkg.in/yaml.v3"
)

func main() {
	// Read configuration
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
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
		handler.SaveHandler(config.Save.Dir),
		handler.CursorCodeHandler(),
		handlers.NewLogHandler(),
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

	log.Println("listener is running...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Stop all sources
	for _, src := range activeSources {
		if err := src.Stop(); err != nil {
			log.Printf("Error stopping source %s: %v", src.Name(), err)
		}
	}
}
