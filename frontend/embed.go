package frontend

import "embed"

// Assets embeds the compiled Svelte/Vite static assets (dist directory).
//
//go:embed all:dist
var Assets embed.FS
