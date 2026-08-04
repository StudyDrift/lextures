package coursechecklist

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed testdata/web_routes.json
var webRoutesJSON []byte

type webRoutesFixture struct {
	Routes []string `json:"routes"`
}

func loadWebRoutesFixture() (map[string]struct{}, error) {
	var f webRoutesFixture
	if err := json.Unmarshal(webRoutesJSON, &f); err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(f.Routes))
	for _, r := range f.Routes {
		out[r] = struct{}{}
	}
	return out, nil
}

func routeExistsInFixture(route string, routes map[string]struct{}) bool {
	if _, ok := routes[route]; ok {
		return true
	}
	// Allow trailing /* style by checking prefix patterns already normalized.
	for r := range routes {
		if strings.HasPrefix(route, strings.TrimSuffix(r, "/*")) {
			return true
		}
	}
	return false
}

func validateNavTargetRoute(route string, routes map[string]struct{}) error {
	if route == "" {
		return fmt.Errorf("empty route")
	}
	if !routeExistsInFixture(route, routes) {
		return fmt.Errorf("route %q not in web route fixture", route)
	}
	return nil
}
