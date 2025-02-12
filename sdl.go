// GOARCH=arm64 GOOS=linux go build -o text main.go
//sudo apt-get install libsdl2-dev on the OPi
//sdl tag version

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
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/fogleman/gg"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	lineSpacing  = 15.0
	topMargin    = 100.0
	bottomMargin = 100.0
	sideMargin   = 100.0
)

const (
	FBIOGET_VSCREENINFO = 0x4600
)

func main() {
	// Define command line flags
	var (
		text           string
		fontPath       string
		backgroundPath string
		fontColorStr   string
		outputPath     string
		useFramebuffer bool
		useSDL         bool
	)

	flag.StringVar(&text, "text", "", "Text to display")
	flag.StringVar(&fontPath, "font", "", "Path to the font file")
	flag.StringVar(&backgroundPath, "background", "", "Path to the background image file")
	flag.StringVar(&fontColorStr, "font-color", "black", "Font color name (e.g., red, green, blue, yellow)")
	flag.StringVar(&outputPath, "output", "output.jpg", "Output image file path")
	flag.BoolVar(&useFramebuffer, "framebuffer", false, "Use framebuffer output instead of a JPG file")
	flag.BoolVar(&useSDL, "sdl", false, "Use SDL output instead of a JPG file")

	flag.Parse()

	// Validate required flags
	if text == "" || fontPath == "" || backgroundPath == "" {
		fmt.Println("Usage: go run main.go -text=<text> -font=<font-path> -background=<background-path> -font-color=<color-name> -output=<output-path>")
		return
	}

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

	fontSize = calculateDynamicFontSize(dc, text, availableWidth)
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

	if useFramebuffer {
		// Use framebuffer output
		displayOnFramebuffer(dc.Image())
	} else if useSDL {
		// Use SDL output
		initializeSDL(dc.Image())
	} else {
		// Save the resulting image as a JPG file
		saveImage(outputPath, dc.Image())
		fmt.Printf("Image saved to %s\n", outputPath)
	}

	// Save the resulting image as a JPG file
	//saveImage(outputPath, dc.Image())
	//fmt.Printf("Image saved to %s\n", outputPath)
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

func calculateDynamicFontSize(dc *gg.Context, text string, maxWidth float64) float64 {
	maxFontSize := 200.0
	minFontSize := 10.0

	// Start with an initial font size
	fontSize := 100.0

	for {
		// Load the font face with the current font size
		err := dc.LoadFontFace("Orbitron-Regular.ttf", fontSize)
		if err != nil {
			log.Fatal(err)
		}

		// Split the text into lines with the current font size
		lines := splitMultilineText(text, dc, maxWidth)

		// Check if the total height exceeds the available height
		totalHeight := calculateTotalTextHeight(lines, dc)
		if totalHeight > float64(dc.Height()-2*topMargin) {
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

func displayOnFramebuffer(img image.Image) {
	// Specify the framebuffer device file (e.g., /dev/fb0).
	framebufferPath := "/dev/fb0"

	// Open the framebuffer device.
	fb, err := os.OpenFile(framebufferPath, os.O_WRONLY, 0)
	if err != nil {
		fmt.Printf("Error opening framebuffer: %v\n", err)
		return
	}
	defer fb.Close()

	// Get framebuffer information directly using syscall.
	info, err := getFramebufferInfo(framebufferPath)
	if err != nil {
		fmt.Printf("Error getting framebuffer info: %v\n", err)
		return
	}

	// Convert string values to integers
	xRes, err := strconv.Atoi(info.XresVirtual)
	if err != nil {
		fmt.Printf("Error converting XresVirtual to integer: %v\n", err)
		return
	}

	yRes, err := strconv.Atoi(info.YresVirtual)
	if err != nil {
		fmt.Printf("Error converting YresVirtual to integer: %v\n", err)
		return
	}

	// Resize the image to fit the framebuffer resolution.
	resizedImg := resizeImage(img, xRes, yRes)

	// Write the resized image to the framebuffer.
	writeFramebuffer(fb, resizedImg)
	fmt.Println("Image displayed on framebuffer successfully.")
}

func getFramebufferInfo(devicePath string) (*UdevFramebufferInfo, error) {
	var info FbVarScreeninfo
	file, err := os.Open(devicePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open device: %v", err)
	}
	defer file.Close()

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&info)))
	if errno != 0 {
		return nil, fmt.Errorf("Failed to get framebuffer info: %v", errno)
	}

	return &UdevFramebufferInfo{
		XresVirtual: strconv.Itoa(int(info.Xres_virtual)),
		YresVirtual: strconv.Itoa(int(info.Yres_virtual)),
	}, nil
}

type UdevFramebufferInfo struct {
	XresVirtual string
	YresVirtual string
}

type FbVarScreeninfo struct {
	Xres_virtual int
	Yres_virtual int
}

func resizeImage(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(newImg, newImg.Bounds(), img, image.Point{}, draw.Src)
	return newImg
}

func writeFramebuffer(fb *os.File, img image.Image) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			pixel := uint32((r>>8)<<0 | (g>>8)<<8 | (b>>8)<<16 | (a>>8)<<24)
			// Assuming 32-bit color depth
			_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), uintptr(uint32(x)|uint32(y)<<16), uintptr(pixel))
			if errno != 0 {
				fmt.Printf("Error writing pixel to framebuffer: %v\n", errno)
				return
			}
		}
	}
}

func initializeSDL(img image.Image) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatalf("Failed to initialize SDL: %s\n", err)
	}
	defer sdl.Quit()

	var window *sdl.Window
	var renderer *sdl.Renderer
	var texture *sdl.Texture

	if err := sdl.CreateWindowAndRenderer(int32(img.Bounds().Dx()), int32(img.Bounds().Dy()), sdl.WINDOW_SHOWN, &window, &renderer); err != nil {
		log.Fatalf("Failed to create window and renderer: %s\n", err)
	}
	defer window.Destroy()
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, int32(img.Bounds().Dx()), int32(img.Bounds().Dy()))
	if err != nil {
		log.Fatalf("Failed to create texture: %s\n", err)
	}
	defer texture.Destroy()

	// Convert the image to pixel data
	pixels := make([]byte, img.Bounds().Dx()*img.Bounds().Dy()*4)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, a := img.At(x, y).RGBA()
			pixels[(y*img.Bounds().Dx()+x)*4] = byte(b >> 8)
			pixels[(y*img.Bounds().Dx()+x)*4+1] = byte(g >> 8)
			pixels[(y*img.Bounds().Dx()+x)*4+2] = byte(r >> 8)
			pixels[(y*img.Bounds().Dx()+x)*4+3] = byte(a >> 8)
		}
	}

	// Update the texture with the pixel data
	texture.Update(nil, pixels, int(img.Bounds().Dx())*4)

	renderer.Clear()
	renderer.Copy(texture, nil, nil)
	renderer.Present()

	// Wait for a key press before exiting
	for {
		event := sdl.WaitEvent()
		switch event := event.(type) {
		case *sdl.QuitEvent:
			return
		case *sdl.KeyboardEvent:
			if event.Type == sdl.KEYDOWN || event.Type == sdl.KEYUP {
				return
			}
		}
	}
}
