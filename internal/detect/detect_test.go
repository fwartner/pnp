package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// helper to create a file with content inside a temp dir
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ---------- DetectProjectType tests ----------

func TestDetectLaravelWeb_WithJobsDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	writeFile(t, dir, "artisan", "#!/usr/bin/env php\n")
	writeFile(t, dir, "app/Jobs/.gitkeep", "")

	result := DetectProjectType(dir)
	if result.Type != "laravel-web" {
		t.Errorf("expected laravel-web, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectLaravelWeb_WithScheduler(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	writeFile(t, dir, "artisan", "#!/usr/bin/env php\n")
	writeFile(t, dir, "routes/console.php", `<?php\nSchedule::command('inspire')->hourly();`)

	result := DetectProjectType(dir)
	if result.Type != "laravel-web" {
		t.Errorf("expected laravel-web, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectLaravelAPI(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	writeFile(t, dir, "artisan", "#!/usr/bin/env php\n")

	result := DetectProjectType(dir)
	if result.Type != "laravel-api" {
		t.Errorf("expected laravel-api, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectNextjsFullstack(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"next":"14.0.0","prisma":"^5.0.0"}}`)

	result := DetectProjectType(dir)
	if result.Type != "nextjs-fullstack" {
		t.Errorf("expected nextjs-fullstack, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectNextjsFullstack_DevDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"next":"14.0.0"},"devDependencies":{"drizzle-orm":"^0.28.0"}}`)

	result := DetectProjectType(dir)
	if result.Type != "nextjs-fullstack" {
		t.Errorf("expected nextjs-fullstack, got %s", result.Type)
	}
}

func TestDetectNextjsStatic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"next":"14.0.0","react":"18.0.0"}}`)

	result := DetectProjectType(dir)
	if result.Type != "nextjs-static" {
		t.Errorf("expected nextjs-static, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectStrapi(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"@strapi/strapi":"^4.0.0"}}`)

	result := DetectProjectType(dir)
	if result.Type != "strapi" {
		t.Errorf("expected strapi, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectUnknown(t *testing.T) {
	dir := t.TempDir()

	result := DetectProjectType(dir)
	if result.Type != "unknown" {
		t.Errorf("expected unknown, got %s", result.Type)
	}
	if result.Confidence != "low" {
		t.Errorf("expected low confidence, got %s", result.Confidence)
	}
}

// ---------- InferProjectName ----------

func TestInferProjectName(t *testing.T) {
	name := InferProjectName("/some/path/my-cool-app")
	if name != "my-cool-app" {
		t.Errorf("expected my-cool-app, got %s", name)
	}
}

// ---------- InferImageFromGitRemote ----------

func TestInferImageFromGitRemote_SSH(t *testing.T) {
	dir := t.TempDir()
	// Set up a minimal git repo with an SSH remote
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = git@github.com:myorg/myrepo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	writeFile(t, dir, ".git/config", config)

	image := InferImageFromGitRemote(dir, "ghcr.io")
	expected := "ghcr.io/myorg/myrepo"
	if image != expected {
		t.Errorf("expected %s, got %s", expected, image)
	}
}

func TestInferImageFromGitRemote_HTTPS(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/anotherorg/anotherrepo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	writeFile(t, dir, ".git/config", config)

	image := InferImageFromGitRemote(dir, "ghcr.io")
	expected := "ghcr.io/anotherorg/anotherrepo"
	if image != expected {
		t.Errorf("expected %s, got %s", expected, image)
	}
}

func TestInferImageFromGitRemote_NoGit(t *testing.T) {
	dir := t.TempDir()
	image := InferImageFromGitRemote(dir, "ghcr.io")
	if image != "" {
		t.Errorf("expected empty string, got %s", image)
	}
}

// ---------- Additional DetectProjectType tests ----------

func TestDetectLaravelWeb_WithConsoleSchedule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^11.0"}}`)
	writeFile(t, dir, "artisan", "#!/usr/bin/env php\n")
	writeFile(t, dir, "routes/console.php", `<?php

use Illuminate\Support\Facades\Schedule;

Schedule::command('emails:send')->daily();
`)

	result := DetectProjectType(dir)
	if result.Type != "laravel-web" {
		t.Errorf("expected laravel-web, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectNextjsFullstack_AllDBDeps(t *testing.T) {
	dbPackages := []string{
		"prisma",
		"@prisma/client",
		"pg",
		"postgres",
		"typeorm",
		"drizzle-orm",
		"knex",
		"sequelize",
	}

	for _, pkg := range dbPackages {
		t.Run(pkg, func(t *testing.T) {
			dir := t.TempDir()
			content := fmt.Sprintf(`{"dependencies":{"next":"14.0.0","%s":"^1.0.0"}}`, pkg)
			writeFile(t, dir, "package.json", content)

			result := DetectProjectType(dir)
			if result.Type != "nextjs-fullstack" {
				t.Errorf("expected nextjs-fullstack with dep %s, got %s", pkg, result.Type)
			}
			if result.Confidence != "high" {
				t.Errorf("expected high confidence, got %s", result.Confidence)
			}
		})
	}
}

func TestDetect_EmptyPackageJson(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{}`)

	result := DetectProjectType(dir)
	if result.Type != "unknown" {
		t.Errorf("expected unknown, got %s", result.Type)
	}
}

func TestDetect_MalformedPackageJson(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{not valid json!!!`)

	result := DetectProjectType(dir)
	if result.Type != "unknown" {
		t.Errorf("expected unknown, got %s", result.Type)
	}
}

func TestDetect_ComposerWithoutArtisan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	// No artisan file

	result := DetectProjectType(dir)
	if result.Type != "unknown" {
		t.Errorf("expected unknown, got %s", result.Type)
	}
}

// ---------- Additional InferImageFromGitRemote tests ----------

func TestInferImageFromGitRemote_HTTPSWithDotGit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".git/config", `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/acme/webapp.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`)

	image := InferImageFromGitRemote(dir, "ghcr.io")
	expected := "ghcr.io/acme/webapp"
	if image != expected {
		t.Errorf("expected %s, got %s", expected, image)
	}
}

func TestInferImageFromGitRemote_HTTPSWithoutDotGit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".git/config", `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/acme/webapp
	fetch = +refs/heads/*:refs/remotes/origin/*
`)

	image := InferImageFromGitRemote(dir, "ghcr.io")
	expected := "ghcr.io/acme/webapp"
	if image != expected {
		t.Errorf("expected %s, got %s", expected, image)
	}
}

// ---------- Additional InferProjectName tests ----------

func TestInferProjectName_NestedPath(t *testing.T) {
	name := InferProjectName("/home/user/projects/deep/nested/my-app")
	if name != "my-app" {
		t.Errorf("expected my-app, got %s", name)
	}
}

func TestInferProjectName_TrailingSlash(t *testing.T) {
	// filepath.Base handles trailing slash by stripping it first
	name := InferProjectName("/some/path/my-app/")
	if name != "my-app" {
		t.Errorf("expected my-app, got %s", name)
	}
}
