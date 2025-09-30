package assets

import "embed"

//go:embed public/*
var PublicContent embed.FS
