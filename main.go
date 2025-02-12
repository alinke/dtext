// ./dtext2 -stats -stage-label="LEVEL" -stage=5 -shots=100 -hits=75 -ratio=75
// TODO need to handle 1280x390 resolution case
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
	"unsafe"

	"github.com/fogleman/gg"
	"golang.org/x/sys/unix"
)

type DisplayMode int

const (
	ModeDefault DisplayMode = iota
	ModeBarOverlay
	ModeStats
	ModeSymbolOverlay
)

type SeparatorStyle int

const (
	SeparatorSolid SeparatorStyle = iota
	SeparatorDashed
	SeparatorDotted
	SeparatorDoubleDot
)

type DrawParams struct {
	Text             string
	FontPath         string
	FontColor        color.Color
	FontSize         float64
	FontSizeOverride float64
	StageLabel       string
	StageValue       int
	LivesLabel       string
	LivesValue       int
	ShotsValue       int
	HitsValue        int
	RatioValue       int
	SeparatorStyle   SeparatorStyle
	SymbolPNG        string
	Symbol2PNG       string
}

const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOPUT_VSCREENINFO = 0x4601
	FBIOGET_FSCREENINFO = 0x4602
)

var (
	topMargin    = 40.0
	bottomMargin = 40.0
	lineSpacing  = 25.0
	sideMargin   = 100.0
)

var (
	symbolPNG        string
	useSymbolOverlay bool
)

var homeDir string
var debug bool

type fbScreenInfo struct {
	Xres         uint32
	Yres         uint32
	XresVirtual  uint32
	YresVirtual  uint32
	XOffset      uint32
	YOffset      uint32
	BitsPerPixel uint32
	Grayscale    uint32
	Red          fbBitfield
	Green        fbBitfield
	Blue         fbBitfield
	Transp       fbBitfield
	Nonstd       uint32
	Activate     uint32
	Height       uint32
	Width        uint32
	Accel_flags  uint32
	PixClock     uint32
	LeftMargin   uint32
	RightMargin  uint32
	UpperMargin  uint32
	LowerMargin  uint32
	HSyncLen     uint32
	VSyncLen     uint32
	Sync         uint32
	VMode        uint32
	Rotate       uint32
	Reserved     [5]uint32
}

type fbBitfield struct {
	Offset   uint32
	Length   uint32
	MSBRight uint32
}

func main() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		return
	}

	homeDir = currentUser.HomeDir
	var sleepDuration time.Duration = 2 * time.Second

	var (
		text             string
		fontPath         string
		backgroundPath   string
		fontColorStr     string
		outputPath       string
		useFramebuffer   bool
		UseGsho          bool
		BarOverlay       bool
		fontSizeOverride float64
		symbolPNG        string
		symbol2PNG       string
	)

	var (
		statsMode  bool
		stageLabel string
		stageValue int
		livesLabel string
		livesValue int
		shotsValue int
		hitsValue  int
		ratioValue int
	)

	var (
		topMarginFlag    float64
		bottomMarginFlag float64
		sideMarginFlag   float64
		lineSpacingFlag  float64
	)

	var timeout int

	var separatorStr string

	flag.StringVar(&text, "text", "", "Text to display")
	flag.StringVar(&fontPath, "font", homeDir+"/pixelcade/fonts/Orbitron-Regular.ttf", "Path to the font file")
	flag.StringVar(&backgroundPath, "background", homeDir+"/pixelcade/backgrounds/background.jpg", "Path to the background image file")
	flag.StringVar(&fontColorStr, "font-color", "white", "Font color name (e.g., red, green, blue, yellow)")
	flag.StringVar(&outputPath, "output", homeDir+"/pixelcade/dtextout.jpg", "Output image file path")
	flag.BoolVar(&useFramebuffer, "framebuffer", true, "Use framebuffer output instead of gsho")
	flag.BoolVar(&UseGsho, "gsho", false, "Display the image using gsho")
	flag.BoolVar(&BarOverlay, "overlay", false, "Display a bottom overlay bar with text")
	flag.StringVar(&symbolPNG, "symbol", "", "Path to first symbol PNG file for overlay")
	flag.StringVar(&symbol2PNG, "symbol2", "", "Path to optional second symbol PNG file for overlay")
	flag.BoolVar(&useSymbolOverlay, "symbol-overlay", false, "Enable symbol overlay mode")

	flag.BoolVar(&statsMode, "stats", false, "Enable stats display mode")
	flag.StringVar(&stageLabel, "stage-label", "STAGE", "Label for the stage/level column")
	flag.StringVar(&livesLabel, "lives-label", "LIVES", "Label for the lives column")
	flag.IntVar(&livesValue, "lives", -1, "Value for lives (if -1, lives column won't be shown)")
	flag.IntVar(&stageValue, "stage", -1, "Value for stage/level (if -1, stage column won't be shown)")
	flag.IntVar(&shotsValue, "shots", -1, "Value for shots (if -1, shots column won't be shown)")
	flag.IntVar(&hitsValue, "hits", -1, "Value for hits (if -1, hits column won't be shown)")
	flag.IntVar(&ratioValue, "ratio", -1, "Value for ratio percentage (if -1, ratio column won't be shown)")
	flag.StringVar(&separatorStr, "separator", "solid", "Separator style (solid, dashed, dotted, doubledot)")

	flag.Float64Var(&lineSpacingFlag, "line-spacing", 25.0, "Line Spacing")
	flag.Float64Var(&topMarginFlag, "top-margin", 40.0, "Top margin")
	flag.Float64Var(&bottomMarginFlag, "bottom-margin", 40.0, "Bottom margin")
	flag.Float64Var(&sideMarginFlag, "side-margin", 100.0, "Side margin")
	flag.Float64Var(&fontSizeOverride, "font-size", 0.0, "Font size override")
	flag.IntVar(&timeout, "timeout", 0, "Close the window after a set duration")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")

	flag.Parse()

	// Validation checks
	if statsMode {
		if stageValue < 0 && livesValue < 0 && shotsValue < 0 && hitsValue < 0 {
			fmt.Println("Stats mode requires at least one of: -stage, -lives, -shots, or -hits")
			return
		}
	} else if symbolPNG == "" && text == "" {
		fmt.Println("Usage: dtext [-gsho] [-text=<text>] [-font=<font-path>] [-background=<background-path>] " +
			"[-font-color=<color-name>] [-output=<output-path>] [-top-margin=<default 140>] " +
			"[-bottom-margin=<default 100>] [-side-margin=<default 100>] [-line-spacing=<default 15>] " +
			"[-font-size=<to override font size>] [-timeout=<x seconds>] [-symbol=<symbol-png-path>]")
		fmt.Println("\nOr for stats mode:")
		fmt.Println("dtext -stats -stage-label=<label> -stage=<value> -lives-label=<label> -lives=<value> -shots=<value> -hits=<value> -ratio=<value>")
		return
	}

	// Update params with default values for symbol-only mode
	if symbolPNG != "" && text == "" {
		text = " " // Use a space as empty text for symbol-only mode
	}

	// Set margins and spacing
	topMargin = topMarginFlag
	bottomMargin = bottomMarginFlag
	sideMargin = sideMarginFlag
	lineSpacing = lineSpacingFlag

	text = strings.ReplaceAll(text, "\\n", string(10))

	fontColor, err := parseColor(fontColorStr)
	if err != nil {
		fmt.Println("Invalid font color:", err)
		return
	}

	debugPrint("Loading background image from: %s\n", backgroundPath)
	bgImage, err := gg.LoadImage(backgroundPath)
	if err != nil {
		log.Fatal(err)
	}

	// Ensure background is 1920x360
	/* bounds := bgImage.Bounds()
	if bounds.Dx() != 1920 || bounds.Dy() != 360 {
		fmt.Printf("Resizing background from %dx%d to 1920x360\n", bounds.Dx(), bounds.Dy())
		bgImage = resizeImage(bgImage, 1920, 360)
	} */

	dc := gg.NewContextForImage(bgImage)

	var fontSize float64

	mode := determineDisplayMode(statsMode, BarOverlay)

	var separatorStyle SeparatorStyle
	switch strings.ToLower(separatorStr) {
	case "dashed":
		separatorStyle = SeparatorDashed
	case "dotted":
		separatorStyle = SeparatorDotted
	case "doubledot":
		separatorStyle = SeparatorDoubleDot
	default:
		separatorStyle = SeparatorSolid
	}

	params := DrawParams{
		Text:             text,
		FontPath:         fontPath,
		FontColor:        fontColor,
		FontSize:         fontSize,
		FontSizeOverride: fontSizeOverride,
		StageLabel:       stageLabel,
		StageValue:       stageValue,
		LivesLabel:       livesLabel,
		LivesValue:       livesValue,
		ShotsValue:       shotsValue,
		HitsValue:        hitsValue,
		RatioValue:       ratioValue,
		SeparatorStyle:   separatorStyle,
		SymbolPNG:        symbolPNG,
		Symbol2PNG:       symbol2PNG,
	}

	if err := drawDisplay(dc, mode, params); err != nil {
		fmt.Printf("Error drawing display: %v", err)
		return
	}

	if !useFramebuffer { //only need to save the file if we're not using framebuffer
		debugPrint("Saving image to: %s\n", outputPath)
		saveImage(outputPath, dc.Image())
	}

	if useFramebuffer {
		debugPrint("Using framebuffer mode...")
		err := writeToFramebuffer(dc.Image())
		if err != nil {
			fmt.Printf("Error writing to framebuffer: %v\n", err)
			return
		}
		if timeout > 0 {
			time.Sleep(time.Duration(timeout) * time.Second)
		} else {
			time.Sleep(sleepDuration)
		}
	} else {
		time.Sleep(200 * time.Millisecond)
		debugPrint("Displaying image with gsho...")
		displayImageWithGsho(outputPath, sleepDuration)
	}

	os.Exit(0)
}

func writeToFramebuffer(img image.Image) error {
	fbFile, err := os.OpenFile("/dev/fb0", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		return fmt.Errorf("failed to open framebuffer: %v", err)
	}
	defer fbFile.Close()

	fbInfo := &fbScreenInfo{}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fbFile.Fd(),
		FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(fbInfo)))
	if errno != 0 {
		return fmt.Errorf("failed to get screen info: %v", errno)
	}

	screenSize := int(fbInfo.Xres * fbInfo.Yres * fbInfo.BitsPerPixel / 8)
	fbMem, err := unix.Mmap(int(fbFile.Fd()), 0, screenSize,
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("failed to mmap framebuffer: %v", err)
	}
	defer unix.Munmap(fbMem)

	bounds := img.Bounds()
	if bounds.Dx() != int(fbInfo.Xres) || bounds.Dy() != int(fbInfo.Yres) {
		img = resizeImage(img, int(fbInfo.Xres), int(fbInfo.Yres))
		bounds = img.Bounds()
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			offset := (y*int(fbInfo.Xres) + x) * 4
			fbMem[offset+2] = byte(r >> 8)
			fbMem[offset+1] = byte(g >> 8)
			fbMem[offset] = byte(b >> 8)
			fbMem[offset+3] = 0xFF
		}
	}

	return nil
}

func determineDisplayMode(statsMode, barOverlay bool) DisplayMode {
	switch {
	case statsMode:
		return ModeStats
	case barOverlay:
		return ModeBarOverlay
	default:
		return ModeDefault
	}
}

func determineSymbolOverlayMode(dc *gg.Context, symbolPNG string, symbol2PNG string) error {
	if symbolPNG == "" {
		return nil
	}

	return drawSymbolOverlay(dc, symbolPNG, symbol2PNG)
}

func drawSymbolOverlay(dc *gg.Context, symbolPath string, symbol2Path string) error {
	fmt.Printf("Loading symbols. Symbol1: %s, Symbol2: %s\n", symbolPath, symbol2Path)

	// Load the first symbol PNG
	symbol1, err := gg.LoadImage(symbolPath)
	if err != nil {
		return fmt.Errorf("failed to load first symbol image: %v", err)
	}
	fmt.Printf("First symbol loaded successfully. Original size: %dx%d\n",
		symbol1.Bounds().Dx(), symbol1.Bounds().Dy())

	// Resize first symbol to 70x70 if needed
	if symbol1.Bounds().Dx() != 70 || symbol1.Bounds().Dy() != 70 {
		debugPrint("Resizing first symbol to 70x70")
		symbol1 = resizeImage(symbol1, 70, 70)
	}

	// Calculate position for first symbol (bottom right corner)
	x1 := float64(dc.Width() - 70 - 5)  // 5px padding from right
	y1 := float64(dc.Height() - 70 - 5) // 5px padding from bottom
	fmt.Printf("Drawing first symbol at position: x=%f, y=%f\n", x1, y1)

	// Draw the first symbol
	dc.DrawImage(symbol1, int(x1), int(y1))

	// If second symbol is specified, load and draw it
	if symbol2Path != "" {
		symbol2, err := gg.LoadImage(symbol2Path)
		if err != nil {
			return fmt.Errorf("failed to load second symbol image: %v", err)
		}
		fmt.Printf("Second symbol loaded successfully. Original size: %dx%d\n",
			symbol2.Bounds().Dx(), symbol2.Bounds().Dy())

		// Resize second symbol to 70x70 if needed
		if symbol2.Bounds().Dx() != 70 || symbol2.Bounds().Dy() != 70 {
			debugPrint("Resizing second symbol to 70x70")
			symbol2 = resizeImage(symbol2, 70, 70)
		}

		// Calculate position for second symbol (to the left of first symbol)
		x2 := x1 - 70 - 5 // 5px padding between symbols
		y2 := y1          // Same vertical position as first symbol
		fmt.Printf("Drawing second symbol at position: x=%f, y=%f\n", x2, y2)

		// Draw the second symbol
		dc.DrawImage(symbol2, int(x2), int(y2))
	}

	return nil
}

// Draw bar overlay mode
func drawBarOverlay(dc *gg.Context, text, fontPath string) error {
	width := dc.Width()
	height := dc.Height()
	barHeight := int(float64(height) * 0.10)

	maxFontSize := calculateMaxFontSize(dc, text, float64(width), float64(barHeight), fontPath)
	fontFace, err := gg.LoadFontFace(fontPath, maxFontSize)
	if err != nil {
		return fmt.Errorf("failed to load font: %v", err)
	}

	dc.SetFontFace(fontFace)
	textWidth, textHeight := dc.MeasureString(text)
	margin := 15.0
	textX := (float64(dc.Width()) - textWidth) / 2.0
	textY := float64(dc.Height()) - (textHeight+10+margin)/2.0

	// Draw black background bar
	dc.DrawRectangle(textX-margin, textY-margin, textWidth+2*margin, textHeight+10+2*margin)
	dc.SetColor(color.Black)
	dc.Fill()

	// Draw text
	dc.SetColor(color.White)
	dc.DrawStringAnchored(text, textX+textWidth/2, textY, 0.5, 0.5)

	return nil
}

// Draw default mode
func drawDefault(dc *gg.Context, text, fontPath string, fontColor color.Color, fontSize float64, fontSizeOverride float64) error {
	err := dc.LoadFontFace(fontPath, fontSize)
	if err != nil {
		return fmt.Errorf("failed to load font: %v", err)
	}
	dc.SetColor(fontColor)

	availableWidth := float64(dc.Width()) - 2*sideMargin
	availableHeight := float64(dc.Height()) - topMargin - bottomMargin

	if fontSizeOverride > 0 {
		fontSize = fontSizeOverride
	} else {
		fontSize = calculateDynamicFontSize(dc, fontPath, text, availableWidth)
	}

	err = dc.LoadFontFace(fontPath, fontSize)
	if err != nil {
		return fmt.Errorf("failed to load font with size: %v", err)
	}
	dc.SetColor(fontColor)

	// Split text into lines
	paragraphs := strings.Split(text, "\n")
	var lines []string
	for _, paragraph := range paragraphs {
		lines = append(lines, splitMultilineText(paragraph, dc, availableWidth)...)
	}

	// Calculate positions and draw
	totalTextHeight := float64(len(lines)-1)*lineSpacing + calculateTotalTextHeight(lines, dc)
	startingY := topMargin + (availableHeight-totalTextHeight)/2

	for _, line := range lines {
		_, h := dc.MeasureString(line)
		dc.DrawStringAnchored(line, float64(dc.Width())/2, startingY, 0.5, 0.5)
		startingY += h + lineSpacing
	}

	return nil
}

func drawDisplay(dc *gg.Context, mode DisplayMode, params DrawParams) error {
	// Draw the base display first
	var err error
	switch mode {
	case ModeBarOverlay:
		err = drawBarOverlay(dc, params.Text, params.FontPath)
	case ModeStats:
		err = drawStats(dc, params.StageLabel, params.StageValue, params.LivesLabel, params.LivesValue,
			params.ShotsValue, params.HitsValue, params.RatioValue,
			params.FontPath, params.SeparatorStyle)
	case ModeDefault:
		err = drawDefault(dc, params.Text, params.FontPath, params.FontColor, params.FontSize, params.FontSizeOverride)
	}

	if err != nil {
		return fmt.Errorf("error drawing base display: %v", err)
	}

	// Always check for symbol overlay after drawing the base display
	if params.SymbolPNG != "" {
		debugPrint("Processing symbol overlay...")
		if err := drawSymbolOverlay(dc, params.SymbolPNG, params.Symbol2PNG); err != nil {
			return fmt.Errorf("error drawing symbol overlay: %v", err)
		}
	}

	return nil
}

func drawStats(dc *gg.Context, stageLabel string, stageValue int, livesLabel string, livesValue int, shotsValue, hitsValue, ratioValue int, fontPath string, separatorStyle SeparatorStyle) error {
	width := float64(dc.Width())
	height := float64(dc.Height())

	// Determine which columns to show
	hasStage := stageValue >= 0
	hasLives := livesValue >= 0
	hasShots := shotsValue >= 0
	hasHits := hitsValue >= 0
	hasRatio := ratioValue >= 0

	// Count how many columns we're showing
	columnCount := 0
	if hasStage {
		columnCount++
	}
	if hasLives {
		columnCount++
	}
	if hasShots {
		columnCount++
	}
	if hasHits {
		columnCount++
	}
	if hasRatio {
		columnCount++
	}

	if columnCount == 0 {
		return nil
	}

	// Calculate column width and positions
	columnWidth := width / float64(columnCount)
	maxLabelSize := columnWidth * 0.4
	maxValueSize := columnWidth * 0.5

	var currentX float64

	labelFontSize := findOptimalFontSize(dc, "SHOTS", maxLabelSize, fontPath)
	valueFontSize := findOptimalFontSize(dc, "999", maxValueSize, fontPath)

	// Function to draw separator line based on style
	drawSeparator := func(x, startY, endY float64) {
		dc.SetColor(color.White)
		dc.SetLineWidth(2.0)

		switch separatorStyle {
		case SeparatorDashed:
			drawDashedLine(dc, x, startY, x, endY, 10.0)
		case SeparatorDotted:
			drawDottedLine(dc, x, startY, x, endY, 5.0)
		case SeparatorDoubleDot:
			drawDoubleDotsLine(dc, x, startY, x, endY, 5.0)
		default: // Solid
			dc.DrawLine(x, startY, x, endY)
			dc.Stroke()
		}
	}

	// Function to draw a column
	drawColumn := func(label string, value string, singleLine bool) {
		err := dc.LoadFontFace(fontPath, labelFontSize)
		if err != nil {
			log.Printf("Error loading font for label: %v", err)
			return
		}

		if singleLine {
			// Single line format for single-column display
			combinedText := fmt.Sprintf("%s %s", label, value)
			dc.SetColor(color.White)
			dc.DrawStringAnchored(combinedText, currentX+columnWidth/2, height*0.5, 0.5, 0.5)
		} else {
			// Two line format for multi-column display
			dc.SetColor(color.White)
			dc.DrawStringAnchored(label, currentX+columnWidth/2, height*0.3, 0.5, 0.5)

			err = dc.LoadFontFace(fontPath, valueFontSize)
			if err != nil {
				log.Printf("Error loading font for value: %v", err)
				return
			}
			dc.DrawStringAnchored(value, currentX+columnWidth/2, height*0.7, 0.5, 0.5)
		}

		if currentX+columnWidth < width {
			drawSeparator(currentX+columnWidth, height*0.1, height*0.9)
		}

		currentX += columnWidth
	}

	// Determine if we should use single-line format (only one value provided)
	singleLine := columnCount == 1

	// Draw columns in specific order: stage, lives, shots, hits, ratio
	if hasStage {
		drawColumn(stageLabel, fmt.Sprintf("%d", stageValue), singleLine && hasStage)
	}
	if hasLives {
		drawColumn(livesLabel, fmt.Sprintf("%d", livesValue), singleLine && hasLives)
	}
	if hasShots {
		drawColumn("SHOTS", fmt.Sprintf("%d", shotsValue), singleLine && hasShots)
	}
	if hasHits {
		drawColumn("HITS", fmt.Sprintf("%d", hitsValue), singleLine && hasHits)
	}
	if hasRatio {
		drawColumn("RATIO", fmt.Sprintf("%d%%", ratioValue), singleLine && hasRatio)
	}

	return nil
}

func drawDashedLine(dc *gg.Context, x1, y1, x2, y2, dashLength float64) {
	dy := y2 - y1
	dashCount := int(dy / (2 * dashLength))

	for i := 0; i < dashCount; i++ {
		startY := y1 + float64(i*2)*dashLength
		endY := startY + dashLength
		dc.DrawLine(x1, startY, x2, endY)
		dc.Stroke()
	}
}

func drawDottedLine(dc *gg.Context, x1, y1, x2, y2, spacing float64) {
	dy := y2 - y1
	dotCount := int(dy / spacing)

	for i := 0; i < dotCount; i++ {
		y := y1 + float64(i)*spacing
		dc.DrawCircle(x1, y, 1.0)
		dc.Fill()
	}
}

func drawDoubleDotsLine(dc *gg.Context, x1, y1, x2, y2, spacing float64) {
	dy := y2 - y1
	dotCount := int(dy / spacing)

	for i := 0; i < dotCount; i++ {
		y := y1 + float64(i)*spacing
		dc.DrawCircle(x1-2, y, 1.0)
		dc.Fill()
		dc.DrawCircle(x1+2, y, 1.0)
		dc.Fill()
	}
}

func findOptimalFontSize(dc *gg.Context, text string, maxWidth float64, fontPath string) float64 {
	fontSize := 200.0 // Start with a large size
	for fontSize > 10 {
		err := dc.LoadFontFace(fontPath, fontSize)
		if err != nil {
			log.Fatal(err)
		}
		width, _ := dc.MeasureString(text)
		if width <= maxWidth {
			return fontSize
		}
		fontSize -= 5
	}
	return 10 // Minimum size
}

func saveImage(filePath string, img image.Image) { //not used with framebuffer mode
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	err = jpeg.Encode(file, img, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func splitMultilineText(text string, dc *gg.Context, maxWidth float64) []string {
	var lines []string
	paragraphs := strings.Split(text, "\n")

	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		var currentLine string
		var lineWidth float64

		for _, word := range words {
			wordWidth, _ := dc.MeasureString(word)

			if lineWidth+wordWidth > maxWidth && len(currentLine) > 0 {
				lines = append(lines, strings.TrimSpace(currentLine))
				currentLine = word + " "
				lineWidth = wordWidth
			} else {
				currentLine += word + " "
				lineWidth += wordWidth
			}
		}

		if len(currentLine) > 0 {
			lines = append(lines, strings.TrimSpace(currentLine))
		}
	}

	return lines
}

func calculateTotalTextHeight(lines []string, dc *gg.Context) float64 {
	var totalHeight float64
	for _, line := range lines {
		_, h := dc.MeasureString(line)
		totalHeight += h
	}
	return totalHeight
}

func calculateDynamicFontSize(dc *gg.Context, fontPath, text string, maxWidth float64) float64 {
	maxFontSize := 200.0
	minFontSize := 10.0

	// Start with an initial font size
	fontSize := 100.0

	for {
		// Load the font face with the current font size
		err := dc.LoadFontFace(fontPath, fontSize)
		if err != nil {
			log.Fatal(err)
		}

		// Split the text into lines with the current font size
		lines := splitMultilineText(text, dc, maxWidth)

		// Check if the total height exceeds the available height
		totalHeight := calculateTotalTextHeight(lines, dc)
		if totalHeight > float64(dc.Height())-2*topMargin {
			fontSize *= 0.9
		} else {
			break
		}

		// Ensure the font size stays within the specified range
		if fontSize < minFontSize {
			fontSize = minFontSize
			break
		}
		if fontSize > maxFontSize {
			fontSize = maxFontSize
			break
		}
	}

	return fontSize
}

// calculateMaxFontSize calculates the maximum font size that fits the text within the specified width and height.
func calculateMaxFontSize(dc *gg.Context, text string, maxWidth, maxHeight float64, fontPath string) float64 {
	maxFontSize := 1.0
	for {
		// Load a font face with the current maximum font size
		fontFace, err := gg.LoadFontFace(fontPath, maxFontSize)
		if err != nil {
			log.Fatal(err)
		}

		// Set the font face
		dc.SetFontFace(fontFace)

		// Measure the width and height of the text
		textWidth, textHeight := dc.MeasureString(text)

		// Check if the text fits within the specified width and height
		if textWidth > maxWidth || textHeight > maxHeight {
			break
		}

		// Increment the font size
		maxFontSize++
	}

	return maxFontSize
}

func parseColor(colorStr string) (color.Color, error) {
	switch strings.ToLower(colorStr) {
	case "black":
		return color.Black, nil
	case "white":
		return color.White, nil
	case "red":
		return color.RGBA{255, 0, 0, 255}, nil
	case "green":
		return color.RGBA{0, 255, 0, 255}, nil
	case "blue":
		return color.RGBA{0, 0, 255, 255}, nil
	case "yellow":
		return color.RGBA{255, 255, 0, 255}, nil
	case "purple":
		return color.RGBA{128, 0, 128, 1}, nil
	case "orange":
		return color.RGBA{255, 165, 0, 1}, nil
	case "cyan":
		return color.RGBA{0, 255, 255, 1}, nil
	case "magenta":
		return color.RGBA{255, 0, 255, 1}, nil
	default:
		return nil, fmt.Errorf("unsupported color: %s", colorStr)
	}
}

func displayImageWithGsho(imagePath string, sleepDuration_ time.Duration) {

	cmd := exec.Command(homeDir+"/pixelcade/gsho", "-platform", "linuxfb", imagePath) //this only works if full paths are declared!
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting gsho:", err)
	}
	time.Sleep(sleepDuration_) //wait 2 seconds and then kill gsho

	// Kill the command process
	err = cmd.Process.Kill()
	if err != nil {
		fmt.Println("Error killing command:", err)
		return
	}
}

/* func resizeImage(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(newImg, newImg.Bounds(), img, image.Point{}, draw.Src)
	return newImg
} */

func resizeImage(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(newImg, newImg.Bounds(), img, img.Bounds().Min, draw.Over)
	return newImg
}

func debugPrint(format string, a ...interface{}) {
	if debug {
		fmt.Printf(format, a...)
	}
}
