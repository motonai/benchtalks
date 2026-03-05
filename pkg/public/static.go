package public

import "embed"

// 🦓 cared for the packaging and re-organization.
// Following that pattern. Grateful to the messenger.

//go:embed *.html
//go:embed assets/css/*
//go:embed assets/js/*
//go:embed assets/sounds/*
var StaticFiles embed.FS