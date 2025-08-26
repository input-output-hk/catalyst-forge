package main

import (
	"log"
	"net/http"

	cfgpkg "github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/config"
	"github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/server"
	u "github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/util"
)

func main() {
	fc, err := cfgpkg.LoadFromEnv()
	if err != nil {
		log.Fatalf("CONFIG_PATH error: %v", err)
	}
	if fc == nil {
		log.Fatalf("CONFIG_PATH must be set and point to a YAML config with providers")
	}
	if len(fc.Providers) == 0 {
		log.Fatalf("config contains no providers; add at least one under providers:")
	}

	mux, err := server.NewMuxFromFileConfig(fc)
	if err != nil {
		log.Fatalf("build mux: %v", err)
	}
	addr := u.FirstNonEmpty(fc.ListenAddr, ":8080")
	log.Printf("mock-oidc (multi) listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
