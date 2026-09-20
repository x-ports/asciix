package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		if isTerminal(os.Stdout) && isTerminal(os.Stdin) {
			runTUI(".")
		} else {
			usage()
		}
		return
	}
	switch os.Args[1] {
	case "tui":
		dir := "."
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		runTUI(dir)
	case "video":
		runVideo(os.Args[2:])
	case "xfetch":
		runXfetch(os.Args[2:])
	case "modes", "--modes":
		printModes()
	case "help", "-h", "--help":
		usage()
	case "version", "-V", "--version":
		fmt.Println("asciix", version)
	default:
		runConvert(os.Args[1:])
	}
}

func usage() {
	fmt.Printf(`asciix %s — convierte imágenes (y vídeo) a arte ASCII

Uso:
  asciix [opciones] imagen1.png [imagen2.jpg ...]
  asciix tui [directorio]           interfaz interactiva
  asciix video [opciones] video.mp4 reproducir vídeo en la terminal
  asciix xfetch imagen.png [nombre] crear un logo para xfetch
  asciix modes                      lista los estilos disponibles

Opciones:
  -m, --mode <modo>     estilo de salida (por defecto ascii)
  -w, --width <n>       ancho en caracteres (por defecto: ancho de terminal)
      --height <n>      alto en líneas (por defecto: proporcional)
  -o, --out <ruta>      archivo o directorio de salida
  -f, --format <fmt>    term | txt | html | png | jpg (por defecto según extensión)
      --color           fuerza color 24-bit
      --no-color        desactiva el color
      --invert          invierte la luminancia
      --threshold <n>   umbral 0-255 para bloques/braille (por defecto 128)
      --ramp <texto>    rampa de caracteres personalizada (modo ascii)
      --contrast        auto-niveles: mejora imágenes oscuras o planas
      --gamma <n>       corrección gamma (por defecto 1.0; >1 aclara)
      --dither          dithering ordenado: más detalle tonal
      --brightness <n>  brillo (multiplicador; 1.0 = normal, 1.5 aclara)
      --saturation <n>  saturación (multiplicador; 1.0 normal, 1.6 más color)
  -r, --recursive       incluir subdirectorios
      --preset <p>      logo | wallpaper
      --list            solo lista los resultados, no los imprime
  -h, --help            esta ayuda

Opciones para exportar imagen (png/jpg):
      --px <n|WxH>      ancho final o lienzo exacto (p. ej. 3840x2160)
      --font <nombre>   tipografía monoespaciada (por defecto "Adwaita Mono")
      --font-size <n>   tamaño de fuente en puntos (por defecto 12)
      --bg <color>      color de fondo (por defecto #000000)
      --quality <n>     calidad JPEG 1-100 (por defecto 92)
      --letter-spacing <pt>  separación horizontal entre caracteres
      --line-spacing <px>    separación vertical entre líneas
      --cover           recorta para llenar el lienzo (sin bordes negros)

Ejemplos:
  asciix foto.jpg -m braille --color
  asciix *.png -w 80 -o salida/ -f html --color
  asciix logo.png --preset logo -o logo.txt
  asciix retrato.jpg -m blocks -w 160 -o retrato.jpg --px 2560
  asciix paisaje.jpg -m blocks -w 240 -o 4k.jpg --px 3840x2160
  asciix tui ~/Imágenes
`, version)
}

func printModes() {
	fmt.Println("Estilos disponibles:")
	for _, m := range modes {
		fmt.Printf("  %-8s %s\n", m.Name, m.Desc)
	}
}

// parseInterspersed parses flags even when they appear after positional
// arguments, which the standard flag package does not allow.
func parseInterspersed(fs *flag.FlagSet, args []string) error {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if len(a) > 1 && a[0] == '-' {
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if strings.ContainsRune(name, '=') {
				continue
			}
			isBool := false
			if f := fs.Lookup(name); f != nil {
				if bv, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
					isBool = bv.IsBoolFlag()
				}
			}
			if !isBool && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return fs.Parse(append(flags, pos...))
}

func runConvert(args []string) {
	fs := flag.NewFlagSet("asciix", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		o             Options
		outPath       string
		format        string
		forceColor    bool
		noColor       bool
		preset        string
		recursive     bool
		listOnly      bool
		font          string
		fontSize      int
		pxSpec        string
		bg            string
		quality       int
		letterSpacing float64
		lineSpacing   int
		cover         bool
	)

	fs.StringVar(&o.Mode, "m", "ascii", "modo")
	fs.StringVar(&o.Mode, "mode", "ascii", "modo")
	fs.IntVar(&o.Width, "w", 0, "ancho en caracteres")
	fs.IntVar(&o.Width, "width", 0, "ancho en caracteres")
	fs.IntVar(&o.Height, "height", 0, "alto en líneas")
	fs.StringVar(&outPath, "o", "", "salida")
	fs.StringVar(&outPath, "out", "", "salida")
	fs.StringVar(&format, "f", "", "formato")
	fs.StringVar(&format, "format", "", "formato")
	fs.BoolVar(&forceColor, "color", false, "color")
	fs.BoolVar(&noColor, "no-color", false, "sin color")
	fs.BoolVar(&o.Invert, "invert", false, "invertir")
	fs.IntVar(&o.Threshold, "threshold", 128, "umbral")
	fs.StringVar(&o.Ramp, "ramp", "", "rampa")
	fs.BoolVar(&o.Contrast, "contrast", false, "auto-niveles (mejora imágenes oscuras)")
	fs.Float64Var(&o.Gamma, "gamma", 1.0, "corrección gamma (p. ej. 1.4)")
	fs.Float64Var(&o.Clip, "clip", 0.5, "porcentaje de recorte para --contrast")
	fs.BoolVar(&o.Dither, "dither", false, "dithering ordenado (conserva más detalle tonal)")
	fs.Float64Var(&o.Brightness, "brightness", 1.0, "brillo (multiplicador; >1 aclara)")
	fs.Float64Var(&o.Saturation, "saturation", 1.0, "saturación (multiplicador; >1 más color)")
	fs.BoolVar(&recursive, "r", false, "recursivo")
	fs.BoolVar(&recursive, "recursive", false, "recursivo")
	fs.StringVar(&preset, "preset", "", "logo | wallpaper")
	fs.BoolVar(&listOnly, "list", false, "solo listar")
	fs.StringVar(&font, "font", "Adwaita Mono", "tipografía para exportar imagen")
	fs.IntVar(&fontSize, "font-size", 12, "tamaño de fuente en puntos")
	fs.StringVar(&pxSpec, "px", "", "ancho o lienzo: 3840 | 3840x2160")
	fs.StringVar(&bg, "bg", "#000000", "color de fondo al exportar imagen")
	fs.IntVar(&quality, "quality", 92, "calidad JPEG (1-100)")
	fs.Float64Var(&letterSpacing, "letter-spacing", 0, "separación horizontal entre caracteres (pt)")
	fs.IntVar(&lineSpacing, "line-spacing", 0, "separación vertical entre líneas (px)")
	fs.BoolVar(&cover, "cover", false, "rellenar el lienzo recortando (sin bordes)")

	if err := parseInterspersed(fs, args); err != nil {
		os.Exit(2)
	}

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	wasSet := func(names ...string) bool {
		for _, n := range names {
			if set[n] {
				return true
			}
		}
		return false
	}

	// Collect inputs, expanding directories.
	var files []string
	dirInput := false
	for _, arg := range fs.Args() {
		info, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "asciix:", err)
			continue
		}
		if info.IsDir() {
			dirInput = true
			found, err := listImages(arg, recursive)
			if err != nil {
				fmt.Fprintln(os.Stderr, "asciix:", err)
				continue
			}
			files = append(files, found...)
		} else {
			files = append(files, arg)
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "asciix: no hay imágenes de entrada")
		usage()
		os.Exit(2)
	}

	// Apply preset defaults only for flags the user did not set.
	tw, _ := termSize()
	switch preset {
	case "logo":
		if !wasSet("m", "mode") {
			o.Mode = "ascii"
		}
		if !wasSet("w", "width") {
			o.Width = 64
		}
		if !wasSet("color", "no-color") {
			noColor = true
		}
	case "wallpaper":
		if !wasSet("m", "mode") {
			o.Mode = "blocks"
		}
		if !wasSet("w", "width") {
			o.Width = tw
		}
		if !wasSet("color", "no-color") {
			forceColor = true
		}
	}

	if o.Width < 1 {
		o.Width = tw
	}
	if format == "" {
		format = inferFormat(outPath)
	}
	format = strings.ToLower(format)
	if isRasterFormat(format) && outPath == "" {
		fmt.Fprintln(os.Stderr, "asciix: los formatos png/jpg requieren -o <archivo>")
		os.Exit(2)
	}
	o.Color = resolveColor(forceColor, noColor, format)

	if cover {
		if tw, th, err := parseSize(pxSpec); err == nil && tw > 0 && th > 0 {
			o.CropAR = float64(tw) / float64(th)
		}
	}

	writeTo := outPath
	outIsDir := writeTo != "" && (dirInput || len(files) > 1 || isDir(writeTo))
	if outIsDir {
		if err := os.MkdirAll(writeTo, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "asciix:", err)
			os.Exit(1)
		}
	}

	hadError := false
	for _, file := range files {
		img, err := loadImage(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "asciix:", err)
			hadError = true
			continue
		}
		g := Convert(img, o)
		if listOnly {
			fmt.Printf("%s  %dx%d  %s\n", file, g.W, g.H, o.Mode)
			continue
		}

		target := writeTo
		if outIsDir {
			base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			target = filepath.Join(writeTo, base+extFor(format))
		}

		if isRasterFormat(format) {
			if err := g.WriteImage(target, o.Color, font, fontSize, pxSpec, quality, bg, letterSpacing, lineSpacing, cover); err != nil {
				fmt.Fprintln(os.Stderr, "asciix:", err)
				hadError = true
				continue
			}
			fmt.Fprintln(os.Stderr, "escrito:", target)
			continue
		}

		var text string
		switch {
		case format == "html":
			text = g.HTML(filepath.Base(file), o.Color)
		case o.Color:
			text = g.ANSI()
		default:
			text = g.Plain()
		}

		if writeTo == "" {
			if len(files) > 1 {
				fmt.Fprintf(os.Stderr, "\n── %s ──\n", file)
			}
			fmt.Print(text)
			continue
		}

		if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "asciix:", err)
			hadError = true
			continue
		}
		fmt.Fprintln(os.Stderr, "escrito:", target)
	}
	if hadError {
		os.Exit(1)
	}
}

func resolveColor(force, no bool, format string) bool {
	if no {
		return false
	}
	if force {
		return true
	}
	// Auto: color for raster/html output and for terminals on a tty.
	switch format {
	case "png", "jpg", "jpeg", "html":
		return true
	case "txt":
		return false
	default:
		return isTerminal(os.Stdout)
	}
}

func inferFormat(out string) string {
	switch strings.ToLower(filepath.Ext(out)) {
	case ".html", ".htm":
		return "html"
	case ".png":
		return "png"
	case ".jpg", ".jpeg":
		return "jpg"
	case ".txt", ".asc", ".ansi":
		return "txt"
	default:
		if out != "" && !isDir(out) && filepath.Ext(out) == "" {
			return "txt"
		}
		return "term"
	}
}

func extFor(format string) string {
	switch format {
	case "html":
		return ".html"
	case "png":
		return ".png"
	case "jpg", "jpeg":
		return ".jpg"
	default:
		return ".txt"
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
