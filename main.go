package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

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
	maxTPM          = 1000.0
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
	progressLabel   = "진행도"
	wpmLabel        = "분당 타수"
	accuracyLabel   = "정확도"
	inputHint       = "현재 문장을 입력하세요."
	headerTitle     = "Mk.04-Go-TypingGame"
	resetButtonText = "초기화"
)

type metrics struct {
	accuracy float64
	progress float64
	tpm      float64
	correct  int
	typed    int
}

func main() {
	a := app.New()

	customTheme := &fontTheme{base: theme.DefaultTheme(), regular: fontData}
	a.Settings().SetTheme(customTheme)

	w := a.NewWindow(headerTitle)
	w.Resize(fyne.NewSize(900, 600))

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

	// Stats
	progressBar := widget.NewProgressBar()
	progressValue := widget.NewLabel("0%")
	progressValue.Alignment = fyne.TextAlignCenter
	
	wpmBar := widget.NewProgressBar()
	wpmValue := widget.NewLabel("0타/분")
	wpmValue.Alignment = fyne.TextAlignCenter

	accuracyBar := widget.NewProgressBar()
	accuracyValue := widget.NewLabel("100%")
	accuracyValue.Alignment = fyne.TextAlignCenter

	// Input
	input := widget.NewEntry()
	input.SetPlaceHolder(inputHint)

	var start time.Time
	var started bool
	currentIdx := 0

	update := func(text string) {
		originalText := text
		// Strip newlines to keep entry single-line.
		if strings.Contains(text, "\n") {
			text = strings.ReplaceAll(text, "\n", "")
			input.SetText(text)
			input.CursorColumn = len([]rune(text))
		}

		// Logic for stats
		pre := strings.Join(targetLines[:currentIdx], "\n")
		typedTotal := pre
		if currentIdx > 0 {
			typedTotal += "\n"
		}
		typedTotal += text

		if !started && len([]rune(text)) > 0 {
			started = true
			start = time.Now()
		}

		m := calculateMetrics(typedTotal, started, start)
		progressBar.SetValue(m.progress)
		progressValue.SetText(fmt.Sprintf("%.0f%%", m.progress*100))

		accuracyBar.SetValue(m.accuracy)
		accuracyValue.SetText(fmt.Sprintf("%.0f%%", m.accuracy*100))

		wpmBar.SetValue(math.Min(m.tpm/maxTPM, 1))
		wpmValue.SetText(fmt.Sprintf("%.0f타/분", m.tpm))

		cleanInput := text // already stripped of newlines above
		targetLine := targetLines[currentIdx]

		// --- Rich Text Coloring Logic ---
		var segments []widget.RichTextSegment
		inputRunes := []rune(cleanInput)
		targetRunes := []rune(targetLine)

		// 1. Process matched/mismatched part
		for i, r := range targetRunes {
			var colorName fyne.ThemeColorName
			if i < len(inputRunes) {
				if inputRunes[i] == r {
					colorName = theme.ColorNameSuccess // Green
				} else {
					colorName = theme.ColorNameError // Red
				}
			} else {
				colorName = theme.ColorNameForeground // Default
			}

			// Optimization: Group consecutive characters of same color?
			// For simplicity and correctness, let's just make individual segments or small groups.
			// Grouping is better for performance.
			segments = append(segments, &widget.TextSegment{
				Text: string(r),
				Style: widget.RichTextStyle{
					Alignment: fyne.TextAlignCenter,
					SizeName:  theme.SizeNameHeadingText, // Make it big
					ColorName: colorName,
				},
			})
		}
		currentTarget.Segments = segments
		currentTarget.Refresh()

		// Check completion
		if cleanInput == targetLine {
			currentLabel.SetText(fmt.Sprintf("현재 문장 (%d/%d) - [스페이스]나 [엔터]를 눌러 계속", currentIdx+1, len(targetLines)))
			
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
		start = time.Time{}
		started = false
		currentIdx = 0
		currentLabel.SetText(fmt.Sprintf("현재 문장 (1/%d)", len(targetLines)))
		setTargetText(targetLines[0])
		update("")
	})

	// --- Layout Construction ---

	// Stats Card
	statsContainer := container.NewGridWithColumns(3,
		container.NewVBox(widget.NewLabelWithStyle(progressLabel, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), progressValue, progressBar),
		container.NewVBox(widget.NewLabelWithStyle(wpmLabel, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), wpmValue, wpmBar),
		container.NewVBox(widget.NewLabelWithStyle(accuracyLabel, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), accuracyValue, accuracyBar),
	)
	statsCard := widget.NewCard("", "", container.NewPadded(statsContainer))

	// Main Typing Area
	typingContent := container.NewVBox(
		currentLabel,
		widget.NewSeparator(),
		container.NewPadded(currentTarget), // The colorful text
		widget.NewSeparator(),
		input,
	)
	typingCard := widget.NewCard("", "", container.NewPadded(typingContent))

	// Global Container
	mainContainer := container.NewBorder(
		nil,      // Top
		reset,    // Bottom
		nil, nil, // Left, Right
		container.NewVBox( // Center content stacked
			statsCard,
			layout.NewSpacer(),
			typingCard,
			layout.NewSpacer(),
		),
	)

	// Add some outer padding
	w.SetContent(container.NewPadded(mainContainer))
	
	// Initial update to set 0 values
	update("")
	w.ShowAndRun()
}

func calculateMetrics(text string, started bool, start time.Time) metrics {
	typedRunes := []rune(text)
	targetRunes := []rune(strings.TrimSpace(targetText))

	typed := len(typedRunes)
	targetLen := len(targetRunes)

	correct := 0
	for i := 0; i < minInt(typed, targetLen); i++ {
		if typedRunes[i] == targetRunes[i] {
			correct++
		}
	}

	progress := 0.0
	if targetLen > 0 {
		progress = float64(typed) / float64(targetLen)
		if progress > 1 {
			progress = 1
		}
	}

	accuracy := 1.0
	if typed > 0 {
		accuracy = float64(correct) / float64(typed)
	}

	tpm := 0.0
	if started {
		elapsed := time.Since(start).Minutes()
		if elapsed > 0 {
			tpm = float64(typed) / elapsed
		}
	}

	return metrics{
		accuracy: accuracy,
		progress: progress,
		tpm:      tpm,
		correct:  correct,
		typed:    typed,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
