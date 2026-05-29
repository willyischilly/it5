package services

import (
	"strings"

	"github.com/jung-kurt/gofpdf"
)

func pdfTextLines(pdf *gofpdf.Fpdf, text string, width float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{"—"}
	}
	lines := pdf.SplitText(text, width)
	if len(lines) == 0 {
		return []string{"—"}
	}
	return lines
}

// pdfDrawTableRow рисует строку таблицы с переносом текста внутри ячеек.
func pdfDrawTableRow(pdf *gofpdf.Fpdf, widths []float64, aligns []string, texts []string, lineH float64, header bool) {
	x0, y0 := pdf.GetX(), pdf.GetY()
	allLines := make([][]string, len(texts))
	maxLines := 1
	for i, t := range texts {
		allLines[i] = pdfTextLines(pdf, t, widths[i]-2)
		if n := len(allLines[i]); n > maxLines {
			maxLines = n
		}
	}
	rowH := lineH * float64(maxLines)

	_, pageH := pdf.GetPageSize()
	_, _, _, mb := pdf.GetMargins()
	if y0+rowH > pageH-mb {
		pdf.AddPage()
		y0 = pdf.GetY()
		x0 = pdf.GetX()
	}

	if header {
		pdf.SetFillColor(230, 230, 230)
	} else {
		pdf.SetFillColor(255, 255, 255)
	}

	for i, lines := range allLines {
		xi := x0
		for j := 0; j < i; j++ {
			xi += widths[j]
		}
		pdf.Rect(xi, y0, widths[i], rowH, "D")
		for li := 0; li < maxLines; li++ {
			line := ""
			if li < len(lines) {
				line = lines[li]
			}
			pdf.SetXY(xi+1, y0+float64(li)*lineH)
			pdf.CellFormat(widths[i]-2, lineH, line, "", 0, aligns[i], header, 0, "")
		}
	}
	pdf.SetXY(x0, y0+rowH)
}

func writeRowWrap(pdf *gofpdf.Fpdf, font, label, value string, labelW, valueW float64, lineH float64) {
	y0 := pdf.GetY()
	x0 := pdf.GetX()
	pdf.SetFont(font, "B", 11)
	pdf.SetXY(x0, y0)
	pdf.CellFormat(labelW, lineH, label+":", "", 0, "L", false, 0, "")

	pdf.SetFont(font, "", 11)
	lines := pdfTextLines(pdf, value, valueW-2)
	valueX := x0 + labelW
	for i, line := range lines {
		pdf.SetXY(valueX, y0+float64(i)*lineH)
		pdf.CellFormat(valueW, lineH, line, "", 0, "L", false, 0, "")
	}
	n := len(lines)
	if n < 1 {
		n = 1
	}
	pdf.SetXY(x0, y0+float64(n)*lineH)
}
