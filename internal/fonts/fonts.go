// Package fonts resolves Penpot font names to Penpot font ids and measures
// text with the real font metrics, since the Penpot backend measures nothing.
package fonts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// fallbackEm is the average advance per character, in ems, used when a font
// file cannot be fetched.
const fallbackEm = 0.55

const maxFontBytes = 8 << 20

// Face is a font family resolved to Penpot's ids for one weight and style.
type Face struct {
	FontID    string // "sourcesanspro" or "gfont-<slug>"
	Family    string // what Penpot stores in fontFamily
	VariantID string // "regular", "bold", "700", "700italic", ...
	Weight    string
	Style     string // "normal" or "italic"
	builtin   bool
	query     string // Google Fonts family name
}

// Resolve maps a family name ("Inter", "Source Sans Pro") and a weight
// (100..900) to a Penpot face. Source Sans Pro is Penpot's built-in font;
// every other family is a Google font.
func Resolve(family string, weight int, italic bool) (Face, error) {
	family = strings.TrimSpace(family)
	if family == "" {
		family = "Source Sans Pro"
	}
	if weight == 0 {
		weight = 400
	}
	if weight < 100 || weight > 900 || weight%100 != 0 {
		return Face{}, fmt.Errorf("font weight must be 100 to 900 in steps of 100")
	}
	style := "normal"
	if italic {
		style = "italic"
	}
	w := strconv.Itoa(weight)
	if isBuiltin(family) {
		return Face{
			FontID: "sourcesanspro", Family: "sourcesanspro", VariantID: builtinVariant(weight, italic),
			Weight: w, Style: style, builtin: true, query: "Source Sans Pro",
		}, nil
	}
	if !familyRE.MatchString(family) {
		return Face{}, fmt.Errorf("font family %q has characters a Google font name cannot have", family)
	}
	variant := w
	switch {
	case weight == 400 && !italic:
		variant = "regular"
	case weight == 400 && italic:
		variant = "italic"
	case italic:
		variant = w + "italic"
	}
	return Face{
		FontID: "gfont-" + slug(family), Family: family, VariantID: variant,
		Weight: w, Style: style, query: family,
	}, nil
}

var familyRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ]{0,60}$`)

func isBuiltin(family string) bool {
	f := strings.ToLower(strings.ReplaceAll(family, " ", ""))
	return f == "sourcesanspro"
}

// builtinVariant follows Penpot's local-fonts table (common/fonts.cljs).
func builtinVariant(weight int, italic bool) string {
	names := map[int]string{200: "200", 300: "300", 400: "regular", 600: "600", 700: "bold", 900: "black"}
	// Nearest available weight.
	best, dist := 400, math.MaxInt
	for wgt := range names {
		if d := abs(wgt - weight); d < dist || (d == dist && wgt > best) {
			best, dist = wgt, d
		}
	}
	v := names[best]
	switch {
	case !italic:
		return v
	case v == "regular":
		return "italic"
	default:
		return v + "italic"
	}
}

func builtinFile(weight int, italic bool) string {
	files := map[string]string{
		"200": "extralight", "300": "light", "regular": "regular", "600": "semibold",
		"bold": "bold", "black": "black",
	}
	v := builtinVariant(weight, false)
	name := files[v]
	if italic {
		if name == "regular" {
			name = "italic"
		} else {
			name += "italic"
		}
	}
	return "sourcesanspro-" + name + ".ttf"
}

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Measurer measures text with real font files, fetched once and cached on
// disk. Built-in fonts come from the Penpot instance; Google fonts from
// fonts.googleapis.com.
type Measurer struct {
	PenpotBase string
	CacheDir   string
	client     *http.Client
	log        *slog.Logger
	googleCSS  string

	mu    sync.Mutex
	faces map[string]*opentype.Font
	fail  map[string]error
}

// NewMeasurer returns a measurer that caches font files under cacheDir.
func NewMeasurer(penpotBase, cacheDir string, logger *slog.Logger) *Measurer {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Measurer{
		PenpotBase: strings.TrimRight(penpotBase, "/"),
		CacheDir:   cacheDir,
		client:     &http.Client{Timeout: 20 * time.Second},
		log:        logger,
		googleCSS:  "https://fonts.googleapis.com/css",
		faces:      map[string]*opentype.Font{},
		fail:       map[string]error{},
	}
}

// Size is a measured text box.
type Size struct {
	Width    float64
	Height   float64
	Measured bool // false when the fallback ratio was used
}

// Measure returns the box of text (lines split on "\n") at size px, with
// letterSpacing px between glyphs and lineHeight as a multiple of size.
func (m *Measurer) Measure(ctx context.Context, f Face, text string, size, lineHeight, letterSpacing float64) Size {
	lines := strings.Split(text, "\n")
	height := math.Ceil(float64(len(lines)) * size * lineHeight)
	otf, err := m.font(ctx, f)
	if err != nil {
		m.log.Warn("font metrics unavailable, estimating", "family", f.Family, "err", err)
		widest := 0.0
		for _, l := range lines {
			n := float64(len([]rune(l)))
			widest = math.Max(widest, n*size*fallbackEm+math.Max(n-1, 0)*letterSpacing)
		}
		return Size{Width: math.Ceil(widest) + 1, Height: height}
	}
	face, err := opentype.NewFace(otf, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return Size{Width: math.Ceil(float64(len(text)) * size * fallbackEm), Height: height}
	}
	defer func() { _ = face.Close() }()
	widest := 0.0
	for _, l := range lines {
		adv := font.MeasureString(face, l)
		n := float64(len([]rune(l)))
		w := fixedToFloat(adv) + math.Max(n-1, 0)*letterSpacing
		widest = math.Max(widest, w)
	}
	return Size{Width: math.Ceil(widest) + 1, Height: height, Measured: true}
}

// MeasureWrapped is Measure for a box of fixed width: words wrap greedily at
// maxWidth, and the height grows to fit.
func (m *Measurer) MeasureWrapped(ctx context.Context, f Face, text string, size, lineHeight, letterSpacing, maxWidth float64) Size {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if m.Measure(ctx, f, cur+" "+w, size, lineHeight, letterSpacing).Width > maxWidth {
				lines = append(lines, cur)
				cur = w
				continue
			}
			cur += " " + w
		}
		lines = append(lines, cur)
	}
	s := m.Measure(ctx, f, strings.Join(lines, "\n"), size, lineHeight, letterSpacing)
	s.Width = maxWidth
	return s
}

func fixedToFloat(v fixed.Int26_6) float64 { return float64(v) / 64 }

func (m *Measurer) font(ctx context.Context, f Face) (*opentype.Font, error) {
	key := f.query + "|" + f.Weight + "|" + f.Style
	m.mu.Lock()
	defer m.mu.Unlock()
	if otf, ok := m.faces[key]; ok {
		return otf, nil
	}
	if err, ok := m.fail[key]; ok {
		return nil, err
	}
	data, err := m.load(ctx, f)
	if err == nil {
		var otf *opentype.Font
		if otf, err = opentype.Parse(data); err == nil {
			m.faces[key] = otf
			return otf, nil
		}
	}
	m.fail[key] = err
	return nil, err
}

func (m *Measurer) load(ctx context.Context, f Face) ([]byte, error) {
	name := slug(f.query) + "-" + f.Weight + f.Style + ".ttf"
	cached := ""
	if m.CacheDir != "" {
		cached = filepath.Join(m.CacheDir, name)
		if data, err := os.ReadFile(filepath.Clean(cached)); err == nil {
			return data, nil
		}
	}
	src, err := m.sourceURL(ctx, f)
	if err != nil {
		return nil, err
	}
	data, err := m.get(ctx, src, "font/ttf")
	if err != nil {
		return nil, err
	}
	if cached != "" {
		if err := os.MkdirAll(m.CacheDir, 0o750); err == nil {
			_ = os.WriteFile(cached, data, 0o600)
		}
	}
	return data, nil
}

var ttfURLRE = regexp.MustCompile(`url\((https://fonts\.gstatic\.com/[^)\s]+\.ttf)\)`)

func (m *Measurer) sourceURL(ctx context.Context, f Face) (string, error) {
	if f.builtin && m.PenpotBase != "" {
		w, _ := strconv.Atoi(f.Weight)
		return m.PenpotBase + "/fonts/" + builtinFile(w, f.Style == "italic"), nil
	}
	ital := ""
	if f.Style == "italic" {
		ital = "italic"
	}
	q := url.Values{"family": {f.query + ":" + f.Weight + ital}}
	css, err := m.get(ctx, m.googleCSS+"?"+q.Encode(), "text/css")
	if err != nil {
		return "", fmt.Errorf("google fonts has no %q %s %s: %w", f.query, f.Weight, f.Style, err)
	}
	match := ttfURLRE.FindSubmatch(css)
	if match == nil {
		return "", errors.New("google fonts reply has no ttf url")
	}
	return string(match[1]), nil
}

func (m *Measurer) get(ctx context.Context, src, _ string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return nil, err
	}
	// A plain user agent makes Google Fonts answer with TTF, not WOFF2.
	req.Header.Set("User-Agent", "Wget/1.21 (penpot-mcp)")
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFontBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFontBytes {
		return nil, errors.New("font file too large")
	}
	return data, nil
}
