package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func runTUI(dir string) {
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\x1b[2J\x1b[H")
		fmt.Println("asciix — interfaz interactiva")
		fmt.Println("-----------------------------")
		fmt.Println("  1) Convertir imágenes a ASCII")
		fmt.Println("  2) Reproducir un vídeo en ASCII")
		fmt.Println("  3) Crear un logo para xfetch")
		fmt.Println("  4) Salir")
		switch askInt(in, "Opción", 1, 1, 4) {
		case 1:
			runImageTUI(in, dir)
		case 2:
			runVideoTUI(in)
		case 3:
			runXfetchTUI(in)
		case 4:
			return
		}
	}
}

func runImageTUI(in *bufio.Reader, dir string) {
	dir, _ = filepath.Abs(dir)
	for {
		files, err := listImages(dir, false)
		if err != nil {
			fmt.Println("Error leyendo el directorio:", err)
			return
		}
		if len(files) == 0 {
			fmt.Println("No se encontraron imágenes en", dir)
		} else {
			fmt.Printf("Directorio: %s\n\n", dir)
			for i, f := range files {
				fmt.Printf("  %2d) %s\n", i+1, filepath.Base(f))
			}
			fmt.Println("  (d) cambiar de directorio   (q) salir")
		}
		fmt.Println()

		sel := prompt(in, "Selecciona imágenes [todo, 1,3, 2-5]", "todo")
		if strings.EqualFold(sel, "q") || strings.EqualFold(sel, "salir") {
			return
		}
		if strings.EqualFold(sel, "d") || strings.EqualFold(sel, "dir") {
			nd := prompt(in, "Nuevo directorio", dir)
			if nd == "" {
				continue
			}
			abs, _ := filepath.Abs(nd)
			if fs, err := listImages(abs, false); err == nil {
				dir, files = abs, fs
			} else {
				fmt.Println("Error:", err)
			}
			continue
		}
		if len(files) == 0 {
			continue
		}

		chosen := parseSelection(sel, files)
		if len(chosen) == 0 {
			fmt.Print("Selección vacía. Intenta de nuevo.\n\n")
			continue
		}

		// Preset o ajustes manuales de estilo.
		fmt.Print("\nPreset:\n  0) ninguno\n  1) logo (ascii 64, sin color)\n  2) wallpaper (blocks, ancho de terminal, color)\n")
		o := Options{Threshold: 128}
		switch askInt(in, "Preset", 0, 0, 2) {
		case 1:
			o.Mode, o.Width, o.Color = "ascii", 64, false
		case 2:
			tw, _ := termSize()
			o.Mode, o.Width, o.Color = "blocks", tw, true
		default:
			fmt.Print("\nEstilos:\n")
			for i, m := range modes {
				fmt.Printf("  %d) %-8s %s\n", i+1, m.Name, m.Desc)
			}
			o.Mode = modes[askInt(in, "Estilo", 1, 1, len(modes))-1].Name
			o.Width = askInt(in, "Ancho en caracteres", 100, 1, 2000)
			o.Color = askBool(in, "¿Color 24-bit?", o.Mode == "blocks")
		}

		// Ajustes de luz y color.
		fmt.Print("\nAjustes de luz/color:\n  0) ninguno\n  1) automático (buena luz y color)\n  2) personalizado\n")
		switch askInt(in, "Ajustes", 1, 0, 2) {
		case 1:
			o.Contrast, o.Gamma, o.Brightness, o.Saturation = true, 1.4, 1.4, 1.7
		case 2:
			o.Contrast = askBool(in, "¿Auto-contraste?", true)
			o.Gamma = askFloat(in, "Gamma (>1 aclara)", 1.4)
			o.Brightness = askFloat(in, "Brillo (>1 aclara)", 1.4)
			o.Saturation = askFloat(in, "Saturación (>1 más color)", 1.7)
			o.Invert = askBool(in, "¿Invertir?", false)
			o.Dither = askBool(in, "¿Dither (más detalle tonal)?", false)
		}

		fmt.Println("\nAcción:\n  1) previsualizar en la terminal\n  2) guardar en archivos\n  3) ambas")
		action := askInt(in, "Acción", 2, 1, 3)

		if action == 1 || action == 3 {
			for _, f := range chosen {
				img, err := loadImage(f)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				g := Convert(img, o)
				fmt.Printf("\x1b[2J\x1b[H%s\n", f)
				if o.Color {
					fmt.Print(g.ANSI())
				} else {
					fmt.Print(g.Plain())
				}
				prompt(in, "\nPulsa Enter para continuar", "")
			}
		}

		if action == 2 || action == 3 {
			fmt.Print("\nFormato de guardado:\n  1) txt\n  2) html\n  3) txt + html\n  4) png\n  5) jpg\n  6) png + jpg\n")
			fchoice := askInt(in, "Formato", 1, 1, 6)
			pxSpec := ""
			cover := false
			if fchoice >= 4 {
				pxSpec = prompt(in, "Lienzo (ancho o ANCHOxALTO)", "3840x2160")
				cover = askBool(in, "¿Rellenar sin bordes (cover)?", true)
			}
			def := "./asciix-out"
			outDir := prompt(in, "Directorio de salida", def)
			if outDir == "" {
				outDir = def
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			for _, f := range chosen {
				img, err := loadImage(f)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				g := Convert(img, o)
				base := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
				writeTxt := fchoice == 1 || fchoice == 3
				writeHTML := fchoice == 2 || fchoice == 3
				writePNG := fchoice == 4 || fchoice == 6
				writeJPG := fchoice == 5 || fchoice == 6
				if writeTxt {
					p := filepath.Join(outDir, base+".txt")
					os.WriteFile(p, []byte(g.Plain()), 0o644)
					fmt.Println("  ✓", p)
				}
				if writeHTML {
					p := filepath.Join(outDir, base+".html")
					os.WriteFile(p, []byte(g.HTML(base, o.Color)), 0o644)
					fmt.Println("  ✓", p)
				}
				if writePNG {
					p := filepath.Join(outDir, base+".png")
					if err := g.WriteImage(p, o.Color, "Adwaita Mono", 12, pxSpec, 92, "#000000", 0, 0, cover); err != nil {
						fmt.Fprintln(os.Stderr, err)
					} else {
						fmt.Println("  ✓", p)
					}
				}
				if writeJPG {
					p := filepath.Join(outDir, base+".jpg")
					if err := g.WriteImage(p, o.Color, "Adwaita Mono", 12, pxSpec, 92, "#000000", 0, 0, cover); err != nil {
						fmt.Fprintln(os.Stderr, err)
					} else {
						fmt.Println("  ✓", p)
					}
				}
			}
		}

		fmt.Println()
		if !askBool(in, "¿Otra conversión?", false) {
			return
		}
		fmt.Print("\x1b[2J\x1b[H")
	}
}

func runVideoTUI(in *bufio.Reader) {
	fmt.Print("\x1b[2J\x1b[HReproducir vídeo en ASCII\n--------------------------\n")
	file := prompt(in, "Ruta del vídeo", "")
	if file == "" {
		return
	}
	if _, err := os.Stat(file); err != nil {
		fmt.Println("No existe:", file)
		return
	}
	fmt.Print("\nEstilos:\n")
	for i, m := range modes {
		fmt.Printf("  %d) %-8s %s\n", i+1, m.Name, m.Desc)
	}
	mode := modes[askInt(in, "Estilo", 4, 1, len(modes))-1].Name
	width := askInt(in, "Ancho en caracteres", 120, 1, 1000)
	fps := askInt(in, "FPS", 15, 1, 60)
	color := askBool(in, "¿Color 24-bit?", true)
	loop := askBool(in, "¿Repetir en bucle?", false)

	args := []string{file, "-m", mode, "-w", strconv.Itoa(width), "--fps", strconv.Itoa(fps)}
	if color {
		args = append(args, "--color")
	} else {
		args = append(args, "--no-color")
	}
	if loop {
		args = append(args, "--loop")
	}
	fmt.Print("\n(Ctrl+C para salir)\n\n")
	runVideo(args)
}

func runXfetchTUI(in *bufio.Reader) {
	fmt.Print("\x1b[2J\x1b[HCrear logo para xfetch\n----------------------\n")
	img := prompt(in, "Imagen del logo", "")
	if img == "" {
		return
	}
	if _, err := os.Stat(img); err != nil {
		fmt.Println("No existe:", img)
		return
	}
	base := strings.TrimSuffix(filepath.Base(img), filepath.Ext(img))
	name := prompt(in, "Nombre del logo", base)
	if name == "" {
		name = base
	}
	fmt.Print("\nEstilos:\n")
	for i, m := range modes {
		fmt.Printf("  %d) %-8s %s\n", i+1, m.Name, m.Desc)
	}
	mode := modes[askInt(in, "Estilo", 1, 1, len(modes))-1].Name
	width := askInt(in, "Ancho", 48, 1, 300)
	update := askBool(in, "¿Actualizar \"ascii\" en config.jsonc?", false)

	force := false
	home, _ := os.UserHomeDir()
	target := filepath.Join(home, ".config", "xfetch", "logos", name+".txt")
	if _, err := os.Stat(target); err == nil {
		force = askBool(in, "El logo ya existe, ¿sobrescribir?", false)
		if !force {
			fmt.Println("Cancelado.")
			return
		}
	}

	args := []string{img, name, "-m", mode, "-w", strconv.Itoa(width)}
	if force {
		args = append(args, "--force")
	}
	if update {
		args = append(args, "--set")
	}
	runXfetch(args)
}

func prompt(in *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func askInt(in *bufio.Reader, label string, def, min, max int) int {
	for {
		s := prompt(in, label, strconv.Itoa(def))
		n, err := strconv.Atoi(s)
		if err == nil && n >= min && n <= max {
			return n
		}
		fmt.Printf("Introduce un número entre %d y %d.\n", min, max)
	}
}

func askFloat(in *bufio.Reader, label string, def float64) float64 {
	for {
		s := prompt(in, label, strconv.FormatFloat(def, 'f', -1, 64))
		v, err := strconv.ParseFloat(s, 64)
		if err == nil && v > 0 {
			return v
		}
		fmt.Println("Introduce un número mayor que 0.")
	}
}

func askBool(in *bufio.Reader, label string, def bool) bool {
	d := "s"
	if !def {
		d = "n"
	}
	for {
		s := strings.ToLower(prompt(in, label+" (s/n)", d))
		switch s {
		case "s", "si", "sí", "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}

// parseSelection accepts "todo", "1", "1,3,5" or "2-4".
func parseSelection(sel string, files []string) []string {
	sel = strings.TrimSpace(strings.ToLower(sel))
	if sel == "" || sel == "todo" || sel == "all" || sel == "*" {
		return append([]string(nil), files...)
	}
	seen := map[int]bool{}
	var out []string
	for _, part := range strings.Split(sel, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if lo, hi, ok := strings.Cut(part, "-"); ok {
			a, err1 := strconv.Atoi(strings.TrimSpace(lo))
			b, err2 := strconv.Atoi(strings.TrimSpace(hi))
			if err1 != nil || err2 != nil {
				continue
			}
			if a > b {
				a, b = b, a
			}
			for i := a; i <= b; i++ {
				if i >= 1 && i <= len(files) && !seen[i] {
					seen[i] = true
					out = append(out, files[i-1])
				}
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err == nil && n >= 1 && n <= len(files) && !seen[n] {
			seen[n] = true
			out = append(out, files[n-1])
		}
	}
	return out
}
