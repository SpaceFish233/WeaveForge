package main

import (
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"

	"weaveforge/db"
	"weaveforge/internal/agent/character"
	"weaveforge/internal/agent/foreshadow"
	"weaveforge/internal/agent/plotengine"
	"weaveforge/internal/agent/setting"
	"weaveforge/internal/agent/style"
	"weaveforge/internal/config"
	"weaveforge/internal/coordinator"
	"weaveforge/internal/llm"
	"weaveforge/internal/vectordb"
	"weaveforge/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Setup log file
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".weaveforge")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "weaveforge.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		multi := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multi)
		defer logFile.Close()
	}
	log.Printf("WeaveForge starting...")

	if err := db.Init(); err != nil {
		log.Fatalf("Database: %v", err)
	}
	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("Config: %v", err)
	}
	config.DecryptConfig(appConfig)

	chapterService := services.NewChapterService(db.DB)
	volumeService := services.NewVolumeService(db.DB)

	// Build embedder: llamacpp (GGUF) or hash fallback
	// Start with hash embedder as immediate fallback, then try llamacpp async
	hashEmbedder := vectordb.NewLocalEmbedder(0)
	var llamacppServer *vectordb.LlamaCppEmbedder
	embedder := vectordb.Embedder(hashEmbedder)

	if appConfig.Embedding.Engine == "llamacpp" {
		lc := vectordb.NewLlamaCppEmbedder(
			appConfig.Embedding.ServerPath,
			appConfig.Embedding.ModelPath,
			appConfig.Embedding.Port,
		)
		llamacppServer = lc
		fallback := vectordb.NewFallbackEmbedder(hashEmbedder, hashEmbedder)
		embedder = fallback
		log.Printf("Starting llama.cpp server in background...")
		go func() {
			if err := lc.Start(); err != nil {
				log.Printf("llama.cpp unavailable: %v, keeping hash embedder", err)
			} else {
				fallback.SetPrimary(lc)
				log.Printf("llama.cpp embedder ready: %s", appConfig.Embedding.ModelPath)
			}
		}()
	} else {
		log.Printf("Using hash embedder")
	}

	defer func() {
		if llamacppServer != nil {
			llamacppServer.Close()
		}
	}()

	chatClient := llm.NewClient(appConfig.LLM.BaseURL, appConfig.LLM.APIKey)
	vectorStore, err := vectordb.NewSQLiteStore(db.DB, embedder)
	if err != nil {
		log.Fatalf("VectorStore: %v", err)
	}

	chatModel := appConfig.LLM.ChatModel
	settingAgent := setting.NewAgent(vectorStore, embedder, chatClient, chatModel, db.DB)
	styleAgent := style.NewAgent(db.DB, chatClient, chatModel)
	foreshadowAgent := foreshadow.NewAgent(db.DB, chatClient, chatModel)
	plotEngine := plotengine.NewAgent(db.DB, chatClient, chatModel)
	characterAgent := character.NewAgent(db.DB)

	coord := coordinator.New(coordinator.AgentHolders{
		Setting: settingAgent, Style: styleAgent,
		Foreshadow: foreshadowAgent,
	})
	coord.Start()
	app := NewApp(chapterService, volumeService, characterAgent, settingAgent, styleAgent, foreshadowAgent, plotEngine, coord, appConfig, embedder)

	if err := wails.Run(&options.App{
		Title:  "WeaveForge",
		Width:  1400,
		Height: 900,
		AssetServer: &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 35, A: 1},
		OnStartup: app.startup,
		Bind: []interface{}{app},
	}); err != nil {
		log.Fatal(err)
	}
}
