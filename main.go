package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "golang.org/x/image/font/sfnt"
)

//go:embed fonts/NanumGothic-Regular.ttf
var fontData []byte

const (
	targetText = "" +
		"동해물과 백두산이 마르고 닳도록\n" +
		"하느님이 보우하사 우리나라 만세\n" +
		"무궁화 삼천리 화려강산\n" +
		"대한사람 대한으로 길이 보전하세\n" +
		"\n" +
		"남산 위에 저 소나무 철갑을 두른 듯\n" +
		"바람 서리 불변함은 우리 기상일세\n" +
		"무궁화 삼천리 화려강산\n" +
		"대한사람 대한으로 길이 보전하세\n" +
		"\n" +
		"가을 하늘 공활한데 높고 구름 없이\n" +
		"밝은 달은 우리 가슴 일편단심일세\n" +
		"무궁화 삼천리 화려강산\n" +
		"대한사람 대한으로 길이 보전하세\n" +
		"\n" +
		"이 기상과 이 맘으로 충성을 다하여\n" +
		"괴로우나 즐거우나 나라 사랑하세\n" +
		"무궁화 삼천리 화려강산\n" +
		"대한사람 대한으로 길이 보전하세"
	inputHint       = "현재 문장을 입력하세요."
	headerTitle     = "Mk.04-Go-TypingGame"
	resetButtonText = "초기화"
)

func main() {
	a := app.New()

	customTheme := &fontTheme{base: theme.DefaultTheme(), regular: fontData}
	a.Settings().SetTheme(customTheme)

	w := a.NewWindow(headerTitle)
	w.Resize(fyne.NewSize(900, 400)) // Adjusted height since stats are gone

	targetLines := normalizeLines(targetText)

	// UI Components
	currentLabel := widget.NewLabel(fmt.Sprintf("현재 문장 (1/%d)", len(targetLines)))
	currentLabel.Alignment = fyne.TextAlignCenter
	currentLabel.TextStyle = fyne.TextStyle{Bold: true}

	// RichText for Target
	currentTarget := widget.NewRichTextFromMarkdown("")
	currentTarget.Wrapping = fyne.TextWrapWord

	// Helper to set initial text
	setTargetText := func(text string) {
		currentTarget.Segments = []widget.RichTextSegment{
			&widget.TextSegment{
				Text: text,
				Style: widget.RichTextStyle{
					Alignment: fyne.TextAlignCenter,
					SizeName:  theme.SizeNameHeadingText,
					ColorName: theme.ColorNameForeground,
				},
			},
		}
		currentTarget.Refresh()
	}
	setTargetText(targetLines[0])

	// Input
	input := widget.NewEntry()
	input.SetPlaceHolder(inputHint)
	input.TextStyle = fyne.TextStyle{Monospace: true} // Monospace often helps with typing games
	// input.Alignment = fyne.TextAlignCenter // NOTE: Fyne's Entry doesn't support Alignment directly in v2.4.x via struct field easily.
	// But we can wrap it or try to center the content if possible.
	// Actually, `widget.Entry` does not expose text alignment easily.
	// However, we can make the input visually centered by wrapping it in a layout or just keep it left-aligned but centered in the container.
	// Since the user asked for "center alignment for typing", and Entry doesn't support `TextAlignCenter` natively without custom rendering,
	// I will focus on the layout centering.

	currentIdx := 0

	update := func(text string) {
		originalText := text
		// Strip newlines to keep entry single-line.
		if strings.Contains(text, "\n") {
			text = strings.ReplaceAll(text, "\n", "")
			input.SetText(text)
			input.CursorColumn = len([]rune(text))
		}

		cleanInput := text // already stripped of newlines above
		targetLine := targetLines[currentIdx]

		// --- Rich Text Coloring Logic ---
		var segments []widget.RichTextSegment
		inputRunes := []rune(cleanInput)
		targetRunes := []rune(targetLine)

		// Grouping logic for TextSegments
		var currentSegmentText strings.Builder
		var currentColor fyne.ThemeColorName

		flushSegment := func() {
			if currentSegmentText.Len() > 0 {
				segments = append(segments, &widget.TextSegment{
					Text: currentSegmentText.String(),
					Style: widget.RichTextStyle{
						Alignment: fyne.TextAlignCenter,
						SizeName:  theme.SizeNameHeadingText,
						ColorName: currentColor,
					},
				})
				currentSegmentText.Reset()
			}
		}

		for i, r := range targetRunes {
			var nextColor fyne.ThemeColorName
			if i < len(inputRunes) {
				if inputRunes[i] == r {
						nextColor = theme.ColorNameSuccess // Green
					} else {
						nextColor = theme.ColorNameError // Red
					}
			} else {
				nextColor = theme.ColorNameForeground // Default
			}

			if i == 0 {
				currentColor = nextColor
			}

			if nextColor != currentColor {
				flushSegment()
				currentColor = nextColor
			}
			currentSegmentText.WriteRune(r)
		}
		flushSegment() // Flush remaining

		currentTarget.Segments = segments
		currentTarget.Refresh()

		// Check completion (Ignore trailing space for the match check)
		trimmedInput := strings.TrimSpace(cleanInput)
		trimmedTarget := strings.TrimSpace(targetLine)

		if trimmedInput == trimmedTarget {
			currentLabel.SetText(fmt.Sprintf("현재 문장 (%d/%d) - [스페이스]나 [엔터]를 눌러 계속", currentIdx+1, len(targetLines)))

			// Check for triggers: Trailing space or Newline (Enter) in the *original* raw input or current text
			isSpaceTrigger := strings.HasSuffix(text, " ")
			isEnterTrigger := strings.Contains(originalText, "\n")

			if isSpaceTrigger || isEnterTrigger {
				advance(&currentIdx, targetLines, currentLabel, input, setTargetText)
			}
		} else {
			currentLabel.SetText(fmt.Sprintf("현재 문장 (%d/%d)", currentIdx+1, len(targetLines)))
		}
	}

	input.OnChanged = update

	reset := widget.NewButton(resetButtonText, func() {
		input.SetText("")
		currentIdx = 0
		currentLabel.SetText(fmt.Sprintf("현재 문장 (1/%d)", len(targetLines)))
		setTargetText(targetLines[0])
		update("")
	})

	// --- Layout Construction ---

	// Main Typing Area - Center everything vertically and horizontally
	typingContent := container.NewVBox(
		currentLabel,
		layout.NewSpacer(),
		container.NewPadded(currentTarget), // The colorful text
		layout.NewSpacer(),
		// Wrap input in a container to control width if needed, or just let it fill
		container.NewPadded(input),
		layout.NewSpacer(),
	)

	// Since we removed stats, the typing card can take up more space or be centered.
	typingCard := widget.NewCard("", "", container.NewPadded(typingContent))

	// Global Container
	mainContainer := container.NewBorder(
		nil,   // Top
		reset, // Bottom
		nil, nil, // Left, Right
		typingCard, // Center
	)

	// Add some outer padding
	w.SetContent(container.NewPadded(mainContainer))

	// Initial update to set 0 values
	update("")
	w.ShowAndRun()
}

func advance(currentIdx *int, targetLines []string, currentLabel *widget.Label, input *widget.Entry, setTargetText func(string)) {
	if *currentIdx+1 >= len(targetLines) {
		// Completed all lines.
		input.SetText("")
		currentLabel.SetText("완료! (모든 문장 입력)")
		setTargetText(targetLines[len(targetLines)-1])
		return
	}
	*currentIdx++
	input.SetText("")
	currentLabel.SetText(fmt.Sprintf("현재 문장 (%d/%d)", *currentIdx+1, len(targetLines)))
	setTargetText(targetLines[*currentIdx])
}

type fontTheme struct {
	base    fyne.Theme
	regular []byte
}

func normalizeLines(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func (f *fontTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return f.base.Color(n, v)
}

func (f *fontTheme) Font(s fyne.TextStyle) fyne.Resource {
	if len(f.regular) == 0 {
		return f.base.Font(s)
	}
	return fyne.NewStaticResource("NanumGothic-Regular.ttf", f.regular)
}

func (f *fontTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return f.base.Icon(n)
}

func (f *fontTheme) Size(n fyne.ThemeSizeName) float32 {
	return f.base.Size(n)
}