// Package web embeds the browser-based settings application.
package web

import "embed"

// Files contains the settings page and its static assets.
//
//go:embed index.html app.css app.js
var Files embed.FS
