//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5/config"
	"github.com/iwaltgen/magex/dep"
	"github.com/iwaltgen/magex/git"
	"github.com/iwaltgen/magex/script"
	"github.com/iwaltgen/magex/semver"
	"github.com/iwaltgen/magex/spinner"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	packageName = "github.com/keecon/talert"
	targetDir   = "build"
)

type (
	BUILD   mg.Namespace
	RELEASE mg.Namespace
	TEST    mg.Namespace
)

var (
	started   int64
	goCmd     string
	workspace string
)

func init() {
	started = time.Now().Unix()
	goCmd = mg.GoCmd()
	workspace, _ = os.Getwd()
}

// Run test cases
func Test() error {
	mg.Deps(Lint)

	return script.ExecStdout(
		fmt.Sprintf("go test ./... -timeout 10s -cover -json"),
		"tparse -all",
	)
}

// Run lint
func Lint() error {
	return sh.RunV("golangci-lint", "run")
}

// Remove all artifacts
func Clean() error {
	if err := sh.Rm(targetDir); err != nil {
		return fmt.Errorf("failed to remove dir `%s`: %w", targetDir, err)
	}
	return nil
}

// Build platform artifacts
func Build() {
	mg.Deps(Lint, BUILD.Platform)
}

// Build platform artifacts
func (ns BUILD) Platform() error {
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to make dir `%s`: %w", targetDir, err)
	}

	if err := sh.RunWith(ns.buildEnv(), goCmd, ns.buildParameters()...); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	return nil
}

// Build linux artifacts (cross compile)
func (ns BUILD) Linux() error {
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to make dir `%s`: %w", targetDir, err)
	}

	if err := sh.RunWith(ns.buildLinuxEnv(), goCmd, ns.buildParameters()...); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	return nil
}

func (BUILD) buildParameters() []string {
	ldflags := "-ldflags=" +
		"-X $PACKAGE/internal.version=$VERSION " +
		"-X $PACKAGE/internal.commitHash=$COMMIT_HASH " +
		"-X $PACKAGE/internal.buildDate=$BUILD_DATE"

	var tags string
	if gotags := os.Getenv("GITHUB_GO_BUILD_TAGS"); gotags != "" {
		tags = gotags
	}

	return []string{"build", "-trimpath", "-tags", tags, ldflags, "-o", targetDir, "./cmd/..."}
}

func (ns BUILD) buildEnv() map[string]string {
	return map[string]string{
		"CGO_ENABLED": "0",
		"PACKAGE":     packageName,
		"WORKSPACE":   workspace,
		"VERSION":     mustCurrentVersion(),
		"COMMIT_HASH": ns.commitHash(),
		"BUILD_DATE":  fmt.Sprintf("%d", started),
	}
}

func (ns BUILD) buildLinuxEnv() map[string]string {
	env := ns.buildEnv()
	env["GOOS"] = "linux"
	env["GOARCH"] = "amd64"
	return env
}

func (BUILD) commitHash() string {
	if gitsha := os.Getenv("GITHUB_SHA"); gitsha != "" {
		return gitsha
	}

	repo, err := git.NewRepository(".")
	if err != nil {
		panic(err)
	}

	ref, err := repo.Head()
	if err != nil {
		panic(err)
	}

	return ref.Hash().String()
}

// Show current version
func Version() error {
	version, err := semver.LatestTag(".")
	if err != nil {
		return err
	}

	color.Green("version: %s", version)
	return nil
}

func mustCurrentVersion() string {
	if tag := os.Getenv("GITHUB_TAG"); tag != "" {
		return tag
	}

	version, err := semver.LatestTag(".")
	if err != nil {
		panic(err)
	}
	return version
}

// Create tag(release) patch version
func Release() error {
	return RELEASE{}.Patch()
}

// Create tag(release) major version
func (ns RELEASE) Major() error {
	return ns.bump(semver.Major)
}

// Create tag(release) minor version
func (ns RELEASE) Minor() error {
	return ns.bump(semver.Minor)
}

// Create tag(release) patch version
func (ns RELEASE) Patch() error {
	return ns.bump(semver.Patch)
}

func (ns RELEASE) bump(typ semver.BumpType) error {
	currentVersion := mustCurrentVersion()
	nextVersion, err := semver.Bump(currentVersion, typ)
	if err != nil {
		return err
	}

	return ns.release(currentVersion, nextVersion)
}

func (ns RELEASE) release(cv, nv string) error {
	err := git.CreateTag(nv,
		git.WithCreateTagMessage("release "+nv),
		git.WithCreateTagPushProgress(os.Stdout),
		git.WithCreateTagHook(ns.prepareCreateTag(cv, nv)),
	)
	if err == nil {
		color.Green("create tag: %s", nv)
	}
	return err
}

func (RELEASE) prepareCreateTag(cv, nv string) func(*git.Repository) error {
	return func(repo *git.Repository) error {
		files := []string{"README.md"}

		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to repository worktree: %w", err)
		}

		cvn, nvn := cv[1:], nv[1:]
		for _, file := range files {
			if _, err := script.ReadFile(file).Replace(cvn, nvn).WriteFile(file); err != nil {
				return fmt.Errorf("failed to bump version `%s`: %w", file, err)
			}

			if _, err := worktree.Add(file); err != nil {
				return fmt.Errorf("failed to git add command `%s`: %w", file, err)
			}
		}

		hash, err := worktree.Commit("chore: bump version", &git.CommitOptions{})
		if err != nil {
			return fmt.Errorf("failed to repository worktree: %w", err)
		}

		opt := &git.PushOptions{
			RefSpecs: []config.RefSpec{
				config.RefSpec("refs/heads/main:refs/heads/main"),
			},
		}
		if err := repo.Push(opt); err == nil {
			color.Green("create tag preprocessing: %s [%s]", nv, hash.String())
		}
		return err
	}
}

// Install tools
func Setup() error {
	defer spinner.Start(100 * time.Millisecond)()

	pkgs, err := dep.GlobImport("tools/deps.go")
	if err != nil {
		return fmt.Errorf("failed to load package import: %w", err)
	}

	args := []string{"install"}
	args = append(args, pkgs...)
	return sh.RunV(goCmd, args...)
}
