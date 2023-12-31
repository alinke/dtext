//GOOS=linux GOARCH=arm64 go build -o dtext main.go

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

	"github.com/fogleman/gg"
	"github.com/veandco/go-sdl2/sdl"
)

func convertImageToSurface(img image.Image) (*sdl.Surface, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	surface, err := sdl.CreateRGBSurface(0, int32(width), int32(height), 32, 0xFF000000, 0x00FF0000, 0x0000FF00, 0x000000FF)
	if err != nil {
		return nil, err
	}

	pixels := surface.Pixels()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			index := (y*width + x) * 4
			pixels[index] = uint8(r >> 8)
			pixels[index+1] = uint8(g >> 8)
			pixels[index+2] = uint8(b >> 8)
			pixels[index+3] = uint8(a >> 8)
		}
	}

	return surface, nil
}

var (
	topMargin    = 40.0
	bottomMargin = 40.0
	lineSpacing  = 25.0
	sideMargin   = 100.0
)

var homeDir string

func main() {

	// Get information about the current user
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		return
	}

	homeDir = currentUser.HomeDir //so we don't need to hard code gshow
	//var sleepDuration time.Duration = 2 * time.Second

	// Define command line flags
	var (
		text             string
		fontPath         string
		backgroundPath   string
		fontColorStr     string
		outputPath       string
		useFramebuffer   bool
		UseGsho          bool
		fontSizeOverride float64 // New flag for font size override
	)

	// Define margin flags
	var (
		topMarginFlag    float64
		bottomMarginFlag float64
		sideMarginFlag   float64
		lineSpacingFlag  float64
	)

	var timeout int //closes the window after x seconds but by default leave the window open

	flag.StringVar(&text, "text", "", "Text to display")
	flag.StringVar(&fontPath, "font", homeDir+"/pixelcade/fonts/Orbitron-Regular.ttf", "Path to the font file")
	flag.StringVar(&backgroundPath, "background", homeDir+"/pixelcade/backgrounds/background.jpg", "Path to the background image file")
	flag.StringVar(&fontColorStr, "font-color", "white", "Font color name (e.g., red, green, blue, yellow)")
	flag.StringVar(&outputPath, "output", homeDir+"/pixelcade/dtextout.jpg", "Output image file path")
	flag.BoolVar(&useFramebuffer, "framebuffer", false, "Use framebuffer output instead of a JPG file") //this caaus
	flag.BoolVar(&UseGsho, "gsho", false, "Display the image using gsho")

	// Define margin flags
	flag.Float64Var(&lineSpacingFlag, "line-spacing", 25.0, "Line Spacing")
	flag.Float64Var(&topMarginFlag, "top-margin", 40.0, "Top margin")
	flag.Float64Var(&bottomMarginFlag, "bottom-margin", 40.0, "Bottom margin")
	flag.Float64Var(&sideMarginFlag, "side-margin", 100.0, "Side margin")
	flag.Float64Var(&fontSizeOverride, "font-size", 0.0, "Font size override")
	flag.IntVar(&timeout, "timeout", 0, "Close the window after a set duration")

	flag.Parse()

	if text == "" {
		fmt.Println("Usage: dtext -gsho -text=<text> -font=<font-path> -background=<background-path> -font-color=<color-name> -output=<output-path> -top-margin=<default 140> -bottom-margin=<default 100> -side-margin=<default 100> -line-spacing=<default 15> -font-size=<to override font size> -timeout=<x seconds>")
		return
	}

	topMargin = topMarginFlag
	bottomMargin = bottomMarginFlag
	sideMargin = sideMarginFlag
	lineSpacing = lineSpacingFlag

	text = strings.ReplaceAll(text, "\\n", string(10)) // Replace "\\n" with ASCII newline character, if we don't do this then the newline is coming through

	//text = "Now Playing Pacman\nAl 99,999\nEKW 44,000\nAKT 33,222\nAL 22,333\nEWD 22,100\nDAG 20,000\nFOB 19,222\nHEL 18,000\nYED 17,000\nPOP 15,000"

	// Parse font color
	fontColor, err := parseColor(fontColorStr)
	if err != nil {
		fmt.Println("Invalid font color:", err)
		return
	}

	// Read the background image
	bgImage, err := gg.LoadImage(backgroundPath)
	if err != nil {
		log.Fatal(err)
	}
	var fontSize float64
	// Create a new drawing context
	dc := gg.NewContextForImage(bgImage)

	// Set the font and color
	err = dc.LoadFontFace(fontPath, fontSize)
	if err != nil {
		log.Fatal(err)
	}
	dc.SetColor(fontColor)

	// Calculate the available width and height considering margins
	availableWidth := float64(dc.Width()) - 2*sideMargin
	availableHeight := float64(dc.Height()) - topMargin - bottomMargin

	// Set the initial font size and color

	if fontSizeOverride > 0 {
		fontSize = fontSizeOverride
	} else {
		fontSize = calculateDynamicFontSize(dc, fontPath, text, availableWidth)
	}

	err = dc.LoadFontFace(fontPath, fontSize)
	if err != nil {
		log.Fatal(err)
	}
	dc.SetColor(fontColor)

	// Split the text into paragraphs
	paragraphs := strings.Split(text, "\n")

	// Split each paragraph into lines
	var lines []string
	for _, paragraph := range paragraphs {
		lines = append(lines, splitMultilineText(paragraph, dc, availableWidth)...)
	}

	// Calculate the total text height
	totalTextHeight := float64(len(lines)-1)*lineSpacing + calculateTotalTextHeight(lines, dc)

	// Calculate the starting position to center the text vertically
	startingY := topMargin + (availableHeight-totalTextHeight)/2

	// Draw each line at the center
	for _, line := range lines {
		_, h := dc.MeasureString(line)
		dc.DrawStringAnchored(line, float64(dc.Width())/2, startingY, 0.5, 0.5)
		startingY += h + lineSpacing
	}

	//OK, our image is done, let's now display it

	// Convert gg.Context image to SDL surface
	surface, err := convertImageToSurface(dc.Image())
	if err != nil {
		fmt.Println("Error converting image to surface:", err)
		return
	}

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Println("Error initializing SDL:", err)
		return
	}
	defer sdl.Quit()

	// Use the width and height of the loaded image
	windowWidth := int32(dc.Width())
	windowHeight := int32(dc.Height())

	// Create window
	window, err := sdl.CreateWindow("SDL Image Display", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, windowWidth, windowHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		fmt.Println("Error creating window:", err)
		return
	}
	defer window.Destroy()

	// Create renderer
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		fmt.Println("Error creating renderer:", err)
		return
	}
	defer renderer.Destroy()

	// Create texture from surface
	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		fmt.Println("Error creating texture:", err)
		return
	}
	defer texture.Destroy()

	// Clear the renderer
	renderer.Clear()

	// Render the texture
	renderer.Copy(texture, nil, nil)

	// Update the screen
	renderer.Present()

	// Wait for the specified duration to see the window
	if timeout > 0 { //TO DO timeout not acctually display the window (at least on the mac)
		timeoutMillis := uint32(timeout) * 1000
		sdl.Delay(timeoutMillis) // Delay for the specified duration
		sdl.Quit()               // Post a quit event to close the window
	} else {
		// Handle events to keep the window open
		for {
			event := sdl.PollEvent()
			switch event := event.(type) {
			case *sdl.QuitEvent:
				return
			case *sdl.KeyboardEvent:
				if event.Keysym.Sym == sdl.K_ESCAPE {
					return
				}
			}
		}
	}

	/*  old code for using gshow to load image from disk and display it
	//Save the resulting image as a JPG file
	fmt.Printf("Image saved to %s\n", outputPath)
	saveImage(outputPath, dc.Image())
	//let's add a pause here from the time we save the image to the time we open it
	time.Sleep(200 * time.Millisecond)
	if UseGsho {
		displayImageWithGsho(outputPath, sleepDuration)
	} */

	os.Exit(0)
}

func saveImage(filePath string, img image.Image) {
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

func resizeImage(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(newImg, newImg.Bounds(), img, image.Point{}, draw.Src)
	return newImg
}
