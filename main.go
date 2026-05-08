package main

import (
	"embed"
	"log"

	"weaveforge/db"
	"weaveforge/internal/agent/character"
	"weaveforge/internal/agent/foreshadow"
	"weaveforge/internal/agent/inspiration"
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
	var embedder vectordb.Embedder
	if appConfig.Embedding.Engine == "llamacpp" {
		lc := vectordb.NewLlamaCppEmbedder(
			appConfig.Embedding.ServerPath,
			appConfig.Embedding.ModelPath,
			appConfig.Embedding.Port,
			appConfig.VectorDB.Dimension,
		)
		if err := lc.Start(); err != nil {
			log.Printf("llama.cpp unavailable: %v, falling back to hash embedder", err)
			embedder = vectordb.NewLocalEmbedder(appConfig.VectorDB.Dimension)
		} else {
			embedder = lc
			log.Printf("Using llama.cpp embedder: %s", appConfig.Embedding.ModelPath)
		}
	}
	if embedder == nil {
		embedder = vectordb.NewLocalEmbedder(appConfig.VectorDB.Dimension)
		log.Printf("Using hash embedder (dim=%d)", appConfig.VectorDB.Dimension)
	}
	if c, ok := embedder.(*vectordb.LlamaCppEmbedder); ok {
		defer c.Close()
	}

	chatClient := llm.NewClient(appConfig.LLM.BaseURL, appConfig.LLM.APIKey)
	vectorStore, err := vectordb.NewSQLiteStore(db.DB, embedder, appConfig.VectorDB.Dimension)
	if err != nil {
		log.Fatalf("VectorStore: %v", err)
	}

	chatModel := appConfig.LLM.ChatModel
	settingAgent := setting.NewAgent(vectorStore, embedder, chatClient, chatModel, db.DB)
	styleAgent := style.NewAgent(db.DB, chatClient, chatModel)
	inspirationAgent := inspiration.NewAgent(db.DB, vectorStore, embedder, chatClient, chatModel)
	foreshadowAgent := foreshadow.NewAgent(db.DB, chatClient, chatModel)
	plotEngine := plotengine.NewAgent(db.DB, chatClient, chatModel)
	characterAgent := character.NewAgent(db.DB)

	coord := coordinator.New(coordinator.AgentHolders{
		Setting: settingAgent, Style: styleAgent,
		Foreshadow: foreshadowAgent, Inspiration: inspirationAgent,
	})
	app := NewApp(chapterService, volumeService, characterAgent, settingAgent, styleAgent, inspirationAgent, foreshadowAgent, plotEngine, coord, appConfig)

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
