package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var asciiLine = regexp.MustCompile(`(?m)^(\s*"ascii"\s*:\s*)"[^"]*"`)

func runXfetch(args []string) {
	fs := flag.NewFlagSet("asciix xfetch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		width int
		mode  string
		force bool
		set   bool
	)
	fs.IntVar(&width, "w", 48, "ancho en caracteres")
	fs.IntVar(&width, "width", 48, "ancho en caracteres")
	fs.StringVar(&mode, "m", "ascii", "modo")
	fs.StringVar(&mode, "mode", "ascii", "modo")
	fs.BoolVar(&force, "force", false, "sobrescribir el logo")
	fs.BoolVar(&set, "set", false, "actualizar \"ascii\" en config.jsonc")

	if err := parseInterspersed(fs, args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "uso: asciix xfetch imagen.png [nombre] [--set]")
		os.Exit(2)
	}
	imgPath := fs.Arg(0)
	name := strings.TrimSuffix(filepath.Base(imgPath), filepath.Ext(imgPath))
	if fs.NArg() > 1 {
		name = fs.Arg(1)
	}
	name = strings.TrimSuffix(name, ".txt")

	home, _ := os.UserHomeDir()
	logoDir := filepath.Join(home, ".config", "xfetch", "logos")
	if err := os.MkdirAll(logoDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "asciix:", err)
		os.Exit(1)
	}
	target := filepath.Join(logoDir, name+".txt")
	if _, err := os.Stat(target); err == nil && !force {
		fmt.Fprintf(os.Stderr, "asciix: %s ya existe (usa --force para sobrescribir)\n", target)
		os.Exit(1)
	}

	img, err := loadImage(imgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "asciix:", err)
		os.Exit(1)
	}
	o := Options{Mode: mode, Width: width, Threshold: 128}
	g := Convert(img, o)
	if err := os.WriteFile(target, []byte(g.Plain()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "asciix:", err)
		os.Exit(1)
	}
	fmt.Println("logo escrito:", target)

	config := filepath.Join(home, ".config", "xfetch", "config.jsonc")
	if set {
		data, err := os.ReadFile(config)
		if err != nil {
			fmt.Fprintln(os.Stderr, "asciix:", err)
			return
		}
		updated := asciiLine.ReplaceAllString(string(data), `${1}"`+target+`"`)
		if updated == string(data) {
			fmt.Fprintln(os.Stderr, "aviso: no se encontró la clave \"ascii\" en config.jsonc")
			return
		}
		if err := os.WriteFile(config, []byte(updated), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "asciix:", err)
			return
		}
		fmt.Println("config actualizada:", config)
	} else {
		fmt.Printf("para usarlo: añade  \"ascii\": \"%s\"  a %s\n", target, config)
	}
}
