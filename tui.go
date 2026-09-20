package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------ config ---

type uiConfig struct {
	preset   string // none | logo | wallpaper
	mode     string
	width    int
	height   int
	color    bool
	invert   bool
	dither   bool
	recurse  bool
	cover    bool
	contrast bool

	threshold     int
	ramp          string
	gamma         float64
	brightness    float64
	saturation    float64
	clip          float64
	format        string
	outDir        string
	pxSpec        string
	font          string
	fontSize      int
	bg            string
	quality       int
	letterSpacing float64
	lineSpacing   int

	// video
	videoFile string
	fps       int
	loop      bool

	// xfetch
	xfetchImage  string
	xfetchName   string
	updateConfig bool
}

func defaultConfig() *uiConfig {
	return &uiConfig{
		preset: "none", mode: "ascii", width: 100, color: true,
		threshold: 128, gamma: 1.0, brightness: 1.0, saturation: 1.0,
		clip: 0.5, format: "term", outDir: "./asciix-out",
		pxSpec: "3840x2160", font: "Adwaita Mono", fontSize: 12,
		bg: "#000000", quality: 92, cover: true,
		fps: 15, loop: false,
	}
}

func (u *uiConfig) toOptions() Options {
	return Options{
		Mode: u.mode, Width: u.width, Height: u.height, Color: u.color,
		Invert: u.invert, Threshold: u.threshold, Ramp: u.ramp,
		Contrast: u.contrast, Gamma: u.gamma, Brightness: u.brightness,
		Saturation: u.saturation, Clip: u.clip, Dither: u.dither,
	}
}

func (u *uiConfig) applyPreset() {
	switch u.preset {
	case "logo":
		u.mode, u.width, u.color = "ascii", 64, false
		u.format = "txt"
	case "wallpaper":
		tw, _ := termSize()
		u.mode, u.width, u.color = "blocks", tw, true
		u.format, u.cover = "jpg", true
	}
	u.setMode(u.mode)
}

func isModeFont(f string) bool {
	for _, m := range modes {
		if m.Font != "" && m.Font == f {
			return true
		}
	}
	return false
}

// setMode changes the style and switches the export font to the style's
// preferred one (CJK, Nerd Font), restoring the default when leaving them.
func (u *uiConfig) setMode(name string) {
	u.mode = name
	m := modeByName(name)
	switch {
	case m.Font != "":
		u.font = m.Font
	case isModeFont(u.font):
		u.font = "Adwaita Mono"
	}
}

// --------------------------------------------------------------------- app ---

const (
	scrMain = iota
	scrImages
	scrVideo
	scrXfetch
)

type app struct {
	in, out *os.File
	restore func()

	screen int
	cfg    *uiConfig
	dir    string
	status string
	errMsg string

	files   []string
	sel     map[string]bool
	cursor  int
	fileTop int
	optTop  int
	focus   int // images: 0 files, 1 options

	xfetchForce bool
}

func runTUI(dir string) {
	in, out := os.Stdin, os.Stdout
	restore, err := enableRaw(in.Fd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "asciix: the interactive UI needs a terminal (%v)\n", err)
		fmt.Fprintln(os.Stderr, "Use the command-line flags instead, e.g.: asciix image.png -m braille --color")
		return
	}
	a := &app{in: in, out: out, restore: restore, cfg: defaultConfig(), sel: map[string]bool{}}
	a.loadDir(dir)

	altEnter(out)
	defer func() {
		altLeave(out)
		a.restore()
	}()

	for {
		a.draw()
		if a.handle(readKey(in)) {
			return
		}
	}
}

func (a *app) loadDir(dir string) {
	abs, _ := filepath.Abs(dir)
	files, err := listImages(abs, a.cfg.recurse)
	if err != nil {
		a.errMsg = err.Error()
		return
	}
	a.cfg.outDir = filepath.Join(filepath.Dir(abs), "asciix-out")
	a.dir = abs
	a.files = files
	a.sel = map[string]bool{}
	a.cursor, a.fileTop, a.optTop = 0, 0, 0
}

func (a *app) selectedFiles() []string {
	var out []string
	for _, f := range a.files {
		if a.sel[f] {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return a.files
	}
	return out
}

func (a *app) enter(screen int) {
	a.screen, a.cursor, a.fileTop, a.optTop = screen, 0, 0, 0
	a.focus = 0
	switch screen {
	case scrImages:
		a.cursor = 0
	case scrVideo:
		a.cursor = firstSelectable(a.videoOpts())
	case scrXfetch:
		a.cursor = firstSelectable(a.xfetchOpts())
	}
}

// ------------------------------------------------------------------ render ---

func (a *app) draw() {
	w, h := termSize()
	c := newCanvas(w, h)
	if h < 6 {
		fmt.Fprint(a.out, c.render())
		return
	}
	titles := map[int]string{
		scrMain:   "asciix — main menu",
		scrImages: "asciix — images to ASCII",
		scrVideo:  "asciix — video to ASCII",
		scrXfetch: "asciix — xfetch logo",
	}
	c.put(0, 0, titles[a.screen], stTitle)
	c.hline(1, '─', stDim)

	switch a.screen {
	case scrMain:
		a.drawMain(c)
	case scrImages:
		a.drawImages(c)
	case scrVideo:
		a.drawForm(c, a.videoOpts(), 1, 2, w-1, h-3, true)
	case scrXfetch:
		a.drawForm(c, a.xfetchOpts(), 1, 2, w-1, h-3, true)
	}

	if a.errMsg != "" {
		c.put(1, h-2, trunc(a.errMsg, w-2), stAccent)
	} else if a.status != "" {
		c.put(1, h-2, trunc(a.status, w-2), stGood)
	}
	c.put(1, h-1, trunc(a.footer(), w-2), stDim)
	fmt.Fprint(a.out, c.render())
}

func (a *app) footer() string {
	switch a.screen {
	case scrMain:
		return "↑/↓ move   enter select   q quit"
	case scrImages:
		return "tab pane   ↑/↓ move   ←/→ change   space toggle   enter edit/run   a all   n none   d dir   p preview   s save   q back"
	case scrVideo:
		return "↑/↓ move   ←/→ change   space toggle   enter edit/run   q back"
	default:
		return "↑/↓ move   ←/→ change   space toggle   enter edit/run   q back"
	}
}

func (a *app) drawMain(c *canvas) {
	items := []string{"Convert images to ASCII", "Play a video as ASCII", "Create an xfetch logo", "Quit"}
	for i, it := range items {
		st, marker := stNormal, "  "
		if i == a.cursor {
			st, marker = stCursor, "> "
		}
		c.put(3, 3+i, marker+it, st)
	}
	c.put(3, 3+len(items)+1, "asciix turns images and video into ASCII art.", stDim)
}

func (a *app) drawImages(c *canvas) {
	w, h := c.w, c.h
	leftW := w / 3
	if leftW < 24 {
		leftW = 24
	}
	if leftW > 48 {
		leftW = 48
	}
	top, bottom := 2, h-3
	listH := bottom - top

	title := fmt.Sprintf("Files (%d/%d selected)", len(a.sel), len(a.files))
	if a.focus == 0 {
		title = "> " + title
	}
	c.put(1, top, trunc(title, leftW-1), stBold)
	if a.focus == 0 {
		if a.cursor < a.fileTop {
			a.fileTop = a.cursor
		}
		if a.cursor >= a.fileTop+listH {
			a.fileTop = a.cursor - listH + 1
		}
	}
	for i := 0; i < listH && a.fileTop+i < len(a.files); i++ {
		idx := a.fileTop + i
		box := "[ ]"
		if a.sel[a.files[idx]] {
			box = "[x]"
		}
		line := box + " " + filepath.Base(a.files[idx])
		st := stNormal
		switch {
		case idx == a.cursor && a.focus == 0:
			st = stCursor
		case a.sel[a.files[idx]]:
			st = stSelected
		}
		c.put(1, top+1+i, trunc(line, leftW-1), st)
	}
	for y := top; y <= bottom; y++ {
		c.put(leftW, y, "│", stDim)
	}
	a.drawForm(c, a.imageOpts(), leftW+2, top+1, w-leftW-3, listH+1, a.focus == 1)
}

func (a *app) drawForm(c *canvas, opts []opt, x, y, w, h int, active bool) {
	if w < 10 {
		w = 10
	}
	listH := h
	if listH < 1 {
		listH = 1
	}
	if active {
		if a.cursor < a.optTop {
			a.optTop = a.cursor
		}
		if a.cursor >= a.optTop+listH {
			a.optTop = a.cursor - listH + 1
		}
	}
	if a.optTop < 0 {
		a.optTop = 0
	}
	head := "Options"
	if active {
		head = "> Options"
	}
	c.put(x, y-1, trunc(head, w), stBold)
	for i := 0; i < listH && a.optTop+i < len(opts); i++ {
		idx := a.optTop + i
		o := opts[idx]
		if o.sep {
			c.put(x, y+i, trunc("── "+o.label+" "+strings.Repeat("─", w), w), stDim)
			continue
		}
		var line string
		if o.get != nil {
			val := o.get()
			if o.adj != nil {
				val = "‹ " + val + " ›"
			}
			line = fmt.Sprintf("%-18s %s", o.label, val)
		} else {
			line = "  " + o.label
		}
		st := stNormal
		switch {
		case active && idx == a.cursor:
			st = stCursor
		case o.act != nil:
			st = stAccent
		}
		c.put(x, y+i, trunc(line, w), st)
	}
}

func trunc(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}

func firstSelectable(opts []opt) int {
	for i, o := range opts {
		if !o.sep {
			return i
		}
	}
	return 0
}

// ------------------------------------------------------------------- input ---

func (a *app) handle(ev keyEvent) bool {
	switch a.screen {
	case scrMain:
		return a.handleMain(ev)
	case scrImages:
		return a.handleImages(ev)
	default:
		return a.handleForm(ev)
	}
}

func (a *app) handleMain(ev keyEvent) bool {
	const n = 4
	switch ev.kind {
	case kUp:
		a.cursor = (a.cursor + n - 1) % n
	case kDown:
		a.cursor = (a.cursor + 1) % n
	case kRune:
		switch ev.r {
		case 'k':
			a.cursor = (a.cursor + n - 1) % n
		case 'j':
			a.cursor = (a.cursor + 1) % n
		case 'q':
			return true
		}
	case kEnter, kSpace:
		switch a.cursor {
		case 0:
			a.enter(scrImages)
		case 1:
			a.enter(scrVideo)
		case 2:
			a.enter(scrXfetch)
		case 3:
			return true
		}
	case kEsc, kCtrlC:
		return true
	}
	return false
}

func (a *app) handleImages(ev keyEvent) bool {
	opts := a.imageOpts()
	switch ev.kind {
	case kCtrlC:
		return true
	case kEsc:
		a.enter(scrMain)
	case kTab:
		if a.focus == 0 {
			a.focus, a.cursor, a.optTop = 1, firstSelectable(opts), 0
		} else {
			a.focus, a.cursor, a.fileTop = 0, 0, 0
		}
	case kUp:
		a.moveCursor(-1)
	case kDown:
		a.moveCursor(1)
	case kLeft, kRight:
		d := -1
		if ev.kind == kRight {
			d = 1
		}
		if a.focus == 1 && a.cursor < len(opts) && opts[a.cursor].adj != nil {
			opts[a.cursor].adj(d)
		}
	case kSpace:
		if a.focus == 1 && a.cursor < len(opts) && opts[a.cursor].tog != nil {
			opts[a.cursor].tog()
		} else {
			a.toggleFile()
		}
	case kEnter:
		if a.focus == 1 && a.cursor < len(opts) {
			o := opts[a.cursor]
			switch {
			case o.act != nil:
				o.act(a)
			case o.edit != nil:
				o.edit(a)
			case o.tog != nil:
				o.tog()
			}
		} else {
			a.toggleFile()
		}
	case kRune:
		switch ev.r {
		case 'j':
			a.moveCursor(1)
		case 'k':
			a.moveCursor(-1)
		case 'a':
			for _, f := range a.files {
				a.sel[f] = true
			}
		case 'n':
			a.sel = map[string]bool{}
		case 'd':
			if p, ok := promptLine(a.in, a.out, "Directory", a.dir); ok && p != "" {
				a.loadDir(p)
			}
		case 'p':
			a.preview()
		case 's':
			a.convert()
		case 'q':
			a.enter(scrMain)
		}
	}
	return false
}

func (a *app) moveCursor(d int) {
	opts := a.optsFor(a.screen)
	if a.screen != scrImages || a.focus == 1 {
		if len(opts) == 0 {
			return
		}
		for {
			a.cursor += d
			if a.cursor < 0 {
				a.cursor = len(opts) - 1
			}
			if a.cursor >= len(opts) {
				a.cursor = 0
			}
			if !opts[a.cursor].sep {
				return
			}
		}
	}
	if len(a.files) == 0 {
		return
	}
	a.cursor += d
	if a.cursor < 0 {
		a.cursor = len(a.files) - 1
	}
	if a.cursor >= len(a.files) {
		a.cursor = 0
	}
}

func (a *app) optsFor(screen int) []opt {
	switch screen {
	case scrImages:
		return a.imageOpts()
	case scrVideo:
		return a.videoOpts()
	case scrXfetch:
		return a.xfetchOpts()
	}
	return nil
}

func (a *app) toggleFile() {
	if a.focus != 0 || a.cursor >= len(a.files) {
		return
	}
	f := a.files[a.cursor]
	a.sel[f] = !a.sel[f]
}

func (a *app) handleForm(ev keyEvent) bool {
	opts := a.optsFor(a.screen)
	switch ev.kind {
	case kCtrlC:
		return true
	case kEsc:
		a.enter(scrMain)
	case kUp:
		a.moveCursor(-1)
	case kDown:
		a.moveCursor(1)
	case kLeft, kRight:
		d := -1
		if ev.kind == kRight {
			d = 1
		}
		if a.cursor < len(opts) && opts[a.cursor].adj != nil {
			opts[a.cursor].adj(d)
		}
	case kSpace:
		if a.cursor < len(opts) && opts[a.cursor].tog != nil {
			opts[a.cursor].tog()
		}
	case kEnter:
		if a.cursor < len(opts) {
			o := opts[a.cursor]
			switch {
			case o.act != nil:
				o.act(a)
			case o.edit != nil:
				o.edit(a)
			case o.tog != nil:
				o.tog()
			}
		}
	case kRune:
		switch ev.r {
		case 'j':
			a.moveCursor(1)
		case 'k':
			a.moveCursor(-1)
		case 'q':
			a.enter(scrMain)
		}
	}
	return false
}

// -------------------------------------------------------------------- opts ---

type opt struct {
	label string
	sep   bool
	get   func() string
	adj   func(d int)
	tog   func()
	edit  func(a *app)
	act   func(a *app)
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func cycle(cur string, list []string, d int) string {
	idx := 0
	for i, s := range list {
		if s == cur {
			idx = i
			break
		}
	}
	return list[(idx+d+len(list))%len(list)]
}

func (a *app) imageOpts() []opt {
	cfg := a.cfg
	var opts []opt

	opts = append(opts, opt{label: "Preset", sep: true}, opt{
		label: "preset",
		get:   func() string { return cfg.preset },
		adj: func(d int) {
			cfg.preset = cycle(cfg.preset, []string{"none", "logo", "wallpaper"}, d)
			cfg.applyPreset()
		},
	})

	opts = append(opts, opt{label: "Style", sep: true}, opt{
		label: "style",
		get:   func() string { return cfg.mode },
		adj:   func(d int) { cfg.setMode(cycle(cfg.mode, modeNames(), d)) },
	}, opt{
		label: "width (cols)",
		get:   func() string { return strconv.Itoa(cfg.width) },
		adj: func(d int) {
			if cfg.width += d * 10; cfg.width < 1 {
				cfg.width = 1
			}
		},
		edit: func(a *app) { editInt(a, "Width", &cfg.width) },
	}, opt{
		label: "height (0=auto)",
		get:   func() string { return strconv.Itoa(cfg.height) },
		adj: func(d int) {
			if cfg.height += d; cfg.height < 0 {
				cfg.height = 0
			}
		},
		edit: func(a *app) { editInt(a, "Height", &cfg.height) },
	}, opt{
		label: "color",
		get:   func() string { return yesno(cfg.color) },
		tog:   func() { cfg.color = !cfg.color },
	}, opt{
		label: "invert",
		get:   func() string { return yesno(cfg.invert) },
		tog:   func() { cfg.invert = !cfg.invert },
	})

	opts = append(opts, opt{label: "Tone / color filters", sep: true}, opt{
		label: "auto-contrast",
		get:   func() string { return yesno(cfg.contrast) },
		tog:   func() { cfg.contrast = !cfg.contrast },
	}, opt{
		label: "gamma",
		get:   func() string { return fmtFloat(cfg.gamma) },
		adj: func(d int) {
			if cfg.gamma += float64(d) * 0.1; cfg.gamma < 0.1 {
				cfg.gamma = 0.1
			}
		},
		edit: func(a *app) { editFloat(a, "Gamma", &cfg.gamma) },
	}, opt{
		label: "brightness",
		get:   func() string { return fmtFloat(cfg.brightness) },
		adj: func(d int) {
			if cfg.brightness += float64(d) * 0.1; cfg.brightness < 0.1 {
				cfg.brightness = 0.1
			}
		},
		edit: func(a *app) { editFloat(a, "Brightness", &cfg.brightness) },
	}, opt{
		label: "saturation",
		get:   func() string { return fmtFloat(cfg.saturation) },
		adj: func(d int) {
			if cfg.saturation += float64(d) * 0.1; cfg.saturation < 0 {
				cfg.saturation = 0
			}
		},
		edit: func(a *app) { editFloat(a, "Saturation", &cfg.saturation) },
	}, opt{
		label: "clip %",
		get:   func() string { return fmtFloat(cfg.clip) },
		adj: func(d int) {
			if cfg.clip += float64(d) * 0.1; cfg.clip < 0 {
				cfg.clip = 0
			}
		},
		edit: func(a *app) { editFloat(a, "Clip %", &cfg.clip) },
	}, opt{
		label: "dither",
		get:   func() string { return yesno(cfg.dither) },
		tog:   func() { cfg.dither = !cfg.dither },
	}, opt{
		label: "threshold",
		get:   func() string { return strconv.Itoa(cfg.threshold) },
		adj: func(d int) {
			if cfg.threshold += d * 8; cfg.threshold < 0 {
				cfg.threshold = 0
			} else if cfg.threshold > 255 {
				cfg.threshold = 255
			}
		},
		edit: func(a *app) { editInt(a, "Threshold", &cfg.threshold) },
	}, opt{
		label: "ramp",
		get: func() string {
			if cfg.ramp == "" {
				return "(default)"
			}
			return cfg.ramp
		},
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Ramp (empty = default)", cfg.ramp); ok {
				cfg.ramp = s
			}
		},
	})

	opts = append(opts, opt{label: "Output", sep: true}, opt{
		label: "format",
		get:   func() string { return cfg.format },
		adj:   func(d int) { cfg.format = cycle(cfg.format, []string{"term", "txt", "html", "png", "jpg"}, d) },
	}, opt{
		label: "output dir",
		get:   func() string { return cfg.outDir },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Output directory", cfg.outDir); ok && s != "" {
				cfg.outDir = s
			}
		},
	}, opt{
		label: "recursive",
		get:   func() string { return yesno(cfg.recurse) },
		tog: func() {
			cfg.recurse = !cfg.recurse
			a.loadDir(a.dir)
		},
	})

	opts = append(opts, opt{label: "Image export (png/jpg)", sep: true}, opt{
		label: "canvas (--px)",
		get:   func() string { return cfg.pxSpec },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Canvas (W or WxH)", cfg.pxSpec); ok {
				cfg.pxSpec = s
			}
		},
	}, opt{
		label: "cover",
		get:   func() string { return yesno(cfg.cover) },
		tog:   func() { cfg.cover = !cfg.cover },
	}, opt{
		label: "font",
		get:   func() string { return cfg.font },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Font", cfg.font); ok && s != "" {
				cfg.font = s
			}
		},
	}, opt{
		label: "font size",
		get:   func() string { return strconv.Itoa(cfg.fontSize) },
		adj: func(d int) {
			if cfg.fontSize += d; cfg.fontSize < 1 {
				cfg.fontSize = 1
			}
		},
		edit: func(a *app) { editInt(a, "Font size", &cfg.fontSize) },
	}, opt{
		label: "background",
		get:   func() string { return cfg.bg },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Background color", cfg.bg); ok && s != "" {
				cfg.bg = s
			}
		},
	}, opt{
		label: "jpeg quality",
		get:   func() string { return strconv.Itoa(cfg.quality) },
		adj: func(d int) {
			if cfg.quality += d * 2; cfg.quality < 1 {
				cfg.quality = 1
			} else if cfg.quality > 100 {
				cfg.quality = 100
			}
		},
		edit: func(a *app) { editInt(a, "JPEG quality", &cfg.quality) },
	}, opt{
		label: "letter spacing",
		get:   func() string { return fmtFloat(cfg.letterSpacing) },
		adj: func(d int) {
			if cfg.letterSpacing += float64(d) * 0.5; cfg.letterSpacing < 0 {
				cfg.letterSpacing = 0
			}
		},
		edit: func(a *app) { editFloat(a, "Letter spacing", &cfg.letterSpacing) },
	}, opt{
		label: "line spacing",
		get:   func() string { return strconv.Itoa(cfg.lineSpacing) },
		adj: func(d int) {
			if cfg.lineSpacing += d; cfg.lineSpacing < -20 {
				cfg.lineSpacing = -20
			}
		},
		edit: func(a *app) { editInt(a, "Line spacing", &cfg.lineSpacing) },
	})

	opts = append(opts, opt{label: "Actions", sep: true},
		opt{label: "Preview selected", act: func(a *app) { a.preview() }},
		opt{label: "Convert / Save selected", act: func(a *app) { a.convert() }},
		opt{label: "Back", act: func(a *app) { a.enter(scrMain) }},
	)
	return opts
}

func (a *app) videoOpts() []opt {
	cfg := a.cfg
	var opts []opt
	opts = append(opts, opt{label: "Video", sep: true}, opt{
		label: "file",
		get:   func() string { return orDash(cfg.videoFile) },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Video file", cfg.videoFile); ok {
				cfg.videoFile = s
			}
		},
	}, opt{
		label: "style",
		get:   func() string { return cfg.mode },
		adj:   func(d int) { cfg.mode = cycle(cfg.mode, modeNames(), d) },
	}, opt{
		label: "width (cols)",
		get:   func() string { return strconv.Itoa(cfg.width) },
		adj: func(d int) {
			if cfg.width += d * 10; cfg.width < 1 {
				cfg.width = 1
			}
		},
	}, opt{
		label: "fps",
		get:   func() string { return strconv.Itoa(cfg.fps) },
		adj: func(d int) {
			if cfg.fps += d; cfg.fps < 1 {
				cfg.fps = 1
			} else if cfg.fps > 60 {
				cfg.fps = 60
			}
		},
		edit: func(a *app) { editInt(a, "FPS", &cfg.fps) },
	}, opt{
		label: "color",
		get:   func() string { return yesno(cfg.color) },
		tog:   func() { cfg.color = !cfg.color },
	}, opt{
		label: "invert",
		get:   func() string { return yesno(cfg.invert) },
		tog:   func() { cfg.invert = !cfg.invert },
	}, opt{
		label: "loop",
		get:   func() string { return yesno(cfg.loop) },
		tog:   func() { cfg.loop = !cfg.loop },
	}, opt{
		label: "threshold",
		get:   func() string { return strconv.Itoa(cfg.threshold) },
		adj: func(d int) {
			if cfg.threshold += d * 8; cfg.threshold < 0 {
				cfg.threshold = 0
			} else if cfg.threshold > 255 {
				cfg.threshold = 255
			}
		},
		edit: func(a *app) { editInt(a, "Threshold", &cfg.threshold) },
	})
	opts = append(opts, opt{label: "Actions", sep: true},
		opt{label: "Play  (Ctrl+C to stop)", act: func(a *app) { a.playVideo() }},
		opt{label: "Back", act: func(a *app) { a.enter(scrMain) }},
	)
	return opts
}

func (a *app) xfetchOpts() []opt {
	cfg := a.cfg
	var opts []opt
	opts = append(opts, opt{label: "xfetch logo", sep: true}, opt{
		label: "image",
		get:   func() string { return orDash(cfg.xfetchImage) },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Image", cfg.xfetchImage); ok {
				cfg.xfetchImage = s
			}
		},
	}, opt{
		label: "name",
		get:   func() string { return orDash(cfg.xfetchName) },
		edit: func(a *app) {
			if s, ok := promptLine(a.in, a.out, "Logo name", cfg.xfetchName); ok {
				cfg.xfetchName = s
			}
		},
	}, opt{
		label: "style",
		get:   func() string { return cfg.mode },
		adj:   func(d int) { cfg.mode = cycle(cfg.mode, modeNames(), d) },
	}, opt{
		label: "width",
		get:   func() string { return strconv.Itoa(cfg.width) },
		adj: func(d int) {
			if cfg.width += d * 4; cfg.width < 1 {
				cfg.width = 1
			}
		},
	}, opt{
		label: "update config.jsonc",
		get:   func() string { return yesno(cfg.updateConfig) },
		tog:   func() { cfg.updateConfig = !cfg.updateConfig },
	})
	opts = append(opts, opt{label: "Actions", sep: true},
		opt{label: "Create logo", act: func(a *app) { a.createXfetch() }},
		opt{label: "Back", act: func(a *app) { a.enter(scrMain) }},
	)
	return opts
}

// -------------------------------------------------------------- edit utils ---

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func orDash(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func editInt(a *app, label string, p *int) {
	if s, ok := promptLine(a.in, a.out, label, strconv.Itoa(*p)); ok {
		if n, err := strconv.Atoi(s); err == nil {
			*p = n
		}
	}
}

func editFloat(a *app, label string, p *float64) {
	if s, ok := promptLine(a.in, a.out, label, fmtFloat(*p)); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			*p = v
		}
	}
}

// ----------------------------------------------------------------- actions ---

func (a *app) preview() {
	files := a.selectedFiles()
	if len(files) == 0 {
		a.status, a.errMsg = "No images selected", ""
		return
	}
	o := a.cfg.toOptions()
	a.leave()
	for _, f := range files {
		img, err := loadImage(f)
		if err != nil {
			fmt.Fprintf(a.out, "\x1b[2J\x1b[H%s\n%v\n", f, err)
		} else {
			g := Convert(img, o)
			fmt.Fprintf(a.out, "\x1b[2J\x1b[H%s\n", f)
			if o.Color {
				fmt.Print(g.ANSI())
			} else {
				fmt.Print(g.Plain())
			}
		}
		fmt.Print("\nPress Enter to continue...")
		for {
			ev := readKey(a.in)
			if ev.kind == kEnter || ev.kind == kEsc || ev.kind == kCtrlC {
				break
			}
		}
	}
	a.reenter()
	a.status = fmt.Sprintf("Previewed %d image(s)", len(files))
	a.errMsg = ""
}

func (a *app) convert() {
	files := a.selectedFiles()
	if len(files) == 0 {
		a.status, a.errMsg = "No images selected", ""
		return
	}
	cfg := a.cfg
	if cfg.format == "term" {
		a.preview()
		return
	}
	o := cfg.toOptions()
	if isRasterFormat(cfg.format) {
		o.Aspect = modeByName(cfg.mode).imgAspect()
	}
	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		a.errMsg = err.Error()
		return
	}
	ok, failed := 0, 0
	for _, f := range files {
		img, err := loadImage(f)
		if err != nil {
			failed++
			continue
		}
		g := Convert(img, o)
		base := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		target := filepath.Join(cfg.outDir, base+extFor(cfg.format))
		switch {
		case isRasterFormat(cfg.format):
			err = g.WriteImage(target, o.Color, cfg.font, cfg.fontSize, cfg.pxSpec, cfg.quality, cfg.bg, cfg.letterSpacing, cfg.lineSpacing, cfg.cover)
		case cfg.format == "html":
			err = os.WriteFile(target, []byte(g.HTML(base, o.Color)), 0o644)
		default:
			err = os.WriteFile(target, []byte(g.Plain()), 0o644)
		}
		if err != nil {
			failed++
		} else {
			ok++
		}
	}
	a.errMsg = ""
	a.status = fmt.Sprintf("Saved %d file(s) to %s (%d failed)", ok, cfg.outDir, failed)
}

func (a *app) playVideo() {
	if a.cfg.videoFile == "" {
		a.status, a.errMsg = "Set a video file first", ""
		return
	}
	if _, err := os.Stat(a.cfg.videoFile); err != nil {
		a.errMsg = "Video not found: " + a.cfg.videoFile
		return
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		a.errMsg = "ffmpeg is required to play video"
		return
	}
	args := []string{a.cfg.videoFile, "-m", a.cfg.mode, "-w", strconv.Itoa(a.cfg.width), "--fps", strconv.Itoa(a.cfg.fps)}
	if a.cfg.color {
		args = append(args, "--color")
	} else {
		args = append(args, "--no-color")
	}
	if a.cfg.invert {
		args = append(args, "--invert")
	}
	if a.cfg.threshold != 128 {
		args = append(args, "--threshold", strconv.Itoa(a.cfg.threshold))
	}
	if a.cfg.loop {
		args = append(args, "--loop")
	}
	a.leave()
	runVideo(args)
	a.reenter()
	a.errMsg = ""
	a.status = "Video stopped"
}

func (a *app) createXfetch() {
	if a.cfg.xfetchImage == "" {
		a.status, a.errMsg = "Set an image first", ""
		return
	}
	if _, err := os.Stat(a.cfg.xfetchImage); err != nil {
		a.errMsg = "Image not found: " + a.cfg.xfetchImage
		return
	}
	name := a.cfg.xfetchName
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(a.cfg.xfetchImage), filepath.Ext(a.cfg.xfetchImage))
	}
	home, _ := os.UserHomeDir()
	logoDir := filepath.Join(home, ".config", "xfetch", "logos")
	target := filepath.Join(logoDir, name+".txt")
	if _, err := os.Stat(target); err == nil && !a.xfetchForce {
		a.xfetchForce = true
		a.status = target + " exists — press Create again to overwrite"
		a.errMsg = ""
		return
	}
	if err := os.MkdirAll(logoDir, 0o755); err != nil {
		a.errMsg = err.Error()
		return
	}
	im, err := loadImage(a.cfg.xfetchImage)
	if err != nil {
		a.errMsg = err.Error()
		return
	}
	o := Options{Mode: a.cfg.mode, Width: a.cfg.width, Threshold: 128}
	g := Convert(im, o)
	if err := os.WriteFile(target, []byte(g.Plain()), 0o644); err != nil {
		a.errMsg = err.Error()
		return
	}
	msg := "Logo written: " + target
	if a.cfg.updateConfig {
		config := filepath.Join(home, ".config", "xfetch", "config.jsonc")
		if data, err := os.ReadFile(config); err == nil {
			updated := asciiLine.ReplaceAllString(string(data), `${1}"`+target+`"`)
			if updated != string(data) {
				os.WriteFile(config, []byte(updated), 0o644)
				msg += " (config.jsonc updated)"
			} else {
				msg += " (could not find \"ascii\" in config.jsonc)"
			}
		}
	}
	a.xfetchForce = false
	a.errMsg = ""
	a.status = msg
}

func (a *app) leave() {
	altLeave(a.out)
	a.restore()
}

func (a *app) reenter() {
	if r, err := enableRaw(a.in.Fd()); err == nil {
		a.restore = r
	}
	altEnter(a.out)
}
