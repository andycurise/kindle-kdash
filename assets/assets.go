package assets

import "embed"

// Fonts contains the font files embedded in every kdash server build.
//
//go:embed fonts/*.ttf
var Fonts embed.FS
