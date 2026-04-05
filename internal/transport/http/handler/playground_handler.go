package handler

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed playground_assets/*
var playgroundAssets embed.FS

type PlaygroundHandler struct {
	assets http.FileSystem
}

func NewPlaygroundHandler() *PlaygroundHandler {
	assets, err := fs.Sub(playgroundAssets, "playground_assets")
	if err != nil {
		panic(err)
	}

	return &PlaygroundHandler{
		assets: http.FS(assets),
	}
}

func (h *PlaygroundHandler) Page(c *gin.Context) {
	body, err := fs.ReadFile(playgroundAssets, "playground_assets/playground.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "playground page not found")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", body)
}

func (h *PlaygroundHandler) AssetsFS() http.FileSystem {
	return h.assets
}
