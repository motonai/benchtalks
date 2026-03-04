package public

import "embed"

//go:embed *.html
//go:embed assets/css/*
//go:embed assets/js/*
//go:embed assets/sounds/*
var StaticFiles embed.FS

