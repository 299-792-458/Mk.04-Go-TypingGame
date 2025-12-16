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
	headerTitle     = "타자 연습: 애국가"
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
	w.Resize(fyne.NewSize(820, 640))

	targetLines := normalizeLines(targetText)
	currentLabel := widget.NewLabel(fmt.Sprintf("현재 문장 (1/%d)", len(targetLines)))
	currentTarget := widget.NewLabelWithStyle(targetLines[0], fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	currentTarget.Wrapping = fyne.TextWrapWord

	progressBar := widget.NewProgressBar()
	progressValue := widget.NewLabel("0%")
	progressRow := container.NewVBox(labelRow(progressLabel, progressValue), progressBar)

	wpmBar := widget.NewProgressBar()
	wpmValue := widget.NewLabel("0타/분")
	wpmRow := container.NewVBox(labelRow(wpmLabel, wpmValue), wpmBar)

	accuracyBar := widget.NewProgressBar()
	accuracyValue := widget.NewLabel("100%")
	accuracyRow := container.NewVBox(labelRow(accuracyLabel, accuracyValue), accuracyBar)

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

		cleanInput := strings.TrimSpace(text)
		currentTargetLine := targetLines[currentIdx]

		// Check if input matches target
		if cleanInput == currentTargetLine {
			// Visual feedback that we are ready to move on
			currentLabel.SetText(fmt.Sprintf("현재 문장 (%d/%d) - [스페이스]나 [엔터]를 눌러 계속", currentIdx+1, len(targetLines)))
			
			// Check for triggers: Trailing space or Newline (Enter)
			isSpaceTrigger := strings.HasSuffix(text, " ")
			isEnterTrigger := strings.Contains(originalText, "\n")

			if isSpaceTrigger || isEnterTrigger {
				advance(&currentIdx, targetLines, currentLabel, currentTarget, input)
			}
		} else {
			// Reset label if user backspaced or is typing
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
		currentTarget.SetText(targetLines[0])
		update("")
	})

	stats := container.NewVBox(progressRow, wpmRow, accuracyRow)
	content := container.NewVBox(
		currentLabel,
		currentTarget,
		widget.NewLabel("타자 입력"),
		input,
		widget.NewSeparator(),
		stats,
		layout.NewSpacer(),
		reset,
	)

	w.SetContent(content)
	update("")
	w.ShowAndRun()
}

func labelRow(title string, value *widget.Label) fyne.CanvasObject {
	return container.NewHBox(
		widget.NewLabel(title),
		layout.NewSpacer(),
		value,
	)
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

func advance(currentIdx *int, targetLines []string, currentLabel *widget.Label, currentTarget *widget.Label, input *widget.Entry) {
	if *currentIdx+1 >= len(targetLines) {
		// Completed all lines.
		input.SetText("")
		currentLabel.SetText("완료! (모든 문장 입력)")
		currentTarget.SetText(targetLines[len(targetLines)-1])
		return
	}
	*currentIdx++
	input.SetText("")
	currentLabel.SetText(fmt.Sprintf("현재 문장 (%d/%d)", *currentIdx+1, len(targetLines)))
	currentTarget.SetText(targetLines[*currentIdx])
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
