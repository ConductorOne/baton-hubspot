package main

import (
	cfg "github.com/conductorone/baton-hubspot/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("hubspot", cfg.Config)
}
