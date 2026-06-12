package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const originalModule = "github.com/rifkifajarramadhani/golang-clean-architecture"

func main() {
	module := flag.String("module", "", "new Go module path")
	service := flag.String("service", "", "service name")
	database := flag.String("database", "", "database name")
	dryRun := flag.Bool("dry-run", false, "show changed files without writing")
	flag.Parse()
	if !validModule(*module) || strings.TrimSpace(*service) == "" || strings.TrimSpace(*database) == "" {
		fmt.Fprintln(os.Stderr, "--module, --service, and --database are required; module must be a valid Go module path")
		os.Exit(2)
	}
	replacements := []string{
		originalModule, *module,
		"go-service", *service,
		"DATABASE_NAME=app", "DATABASE_NAME=" + *database,
		"  name: app", "  name: " + *database,
	}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasPrefix(path, ".git/") || path == ".env" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil || strings.IndexByte(string(data), 0) >= 0 {
			return err
		}
		updated := strings.NewReplacer(replacements...).Replace(string(data))
		if updated == string(data) {
			return nil
		}
		fmt.Println(path)
		if *dryRun {
			return nil
		}
		return os.WriteFile(path, []byte(updated), info.Mode())
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validModule(value string) bool {
	return regexp.MustCompile(`^[A-Za-z0-9._~-]+(?:/[A-Za-z0-9._~-]+)+$`).MatchString(value)
}
