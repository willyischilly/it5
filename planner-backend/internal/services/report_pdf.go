package services

import (
	"bytes"
	"fmt"
	"time"

	"planner-backend/internal/services/pdffonts"

	"github.com/jung-kurt/gofpdf"
)

var reportStatusRu = map[string]string{
	"draft":       "Черновик",
	"submitted":   "Отправлена",
	"in_progress": "В работе",
	"completed":   "Завершена",
	"overdue":     "Просрочена",
}

var taskStatusRu = map[string]string{
	"pending":     "В планах",
	"in_progress": "В работе",
	"completed":   "Завершено",
}

func statusRu(m map[string]string, code string) string {
	if s, ok := m[code]; ok {
		return s
	}
	return code
}

func pdfFont() (fontName string, pdf *gofpdf.Fpdf) {
	pdf = gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	fontName = "Helvetica"
	if path := pdffonts.ArialPath(); path != "" {
		pdf.AddUTF8Font("Arial", "", path)
		pdf.AddUTF8Font("Arial", "B", path)
		fontName = "Arial"
	}
	return fontName, pdf
}

func pdfOutput(pdf *gofpdf.Fpdf) ([]byte, error) {
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func BuildSummaryReportPDF(summary *SummaryReportResponse) ([]byte, error) {
	fontName, pdf := pdfFont()
	pdf.AddPage()

	pdf.SetFont(fontName, "B", 16)
	pdf.CellFormat(0, 10, "Сводный отчёт по заявкам", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont(fontName, "", 11)
	writeRow(pdf, fontName, "Всего заявок", fmt.Sprintf("%d", summary.TotalRequests))
	writeRow(pdf, fontName, "Завершено", fmt.Sprintf("%d", summary.CompletedRequests))
	writeRow(pdf, fontName, "В работе", fmt.Sprintf("%d", summary.InProgressRequests))
	writeRow(pdf, fontName, "Отправлено", fmt.Sprintf("%d", summary.SubmittedRequests))
	writeRow(pdf, fontName, "Черновики", fmt.Sprintf("%d", summary.DraftRequests))
	writeRow(pdf, fontName, "Просрочено", fmt.Sprintf("%d", summary.OverdueRequests))
	pdf.Ln(2)
	writeRow(pdf, fontName, "Задач всего", fmt.Sprintf("%d (выполнено %d)", summary.TotalTasks, summary.CompletedTasks))
	writeRow(pdf, fontName, "Нормочасов", fmt.Sprintf("%d (выполнено %d)", summary.TotalHours, summary.CompletedHours))
	pdf.Ln(6)

	pdf.SetFont(fontName, "B", 12)
	pdf.CellFormat(0, 8, "Заявки", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	colW := []float64{12, 42, 22, 26, 22, 22, 28}
	headers := []string{"№", "Название", "Контур", "Статус", "Задачи", "Часы", "Дедлайн"}
	pdf.SetFont(fontName, "B", 9)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(colW[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont(fontName, "", 9)
	pdf.SetFillColor(255, 255, 255)
	for _, r := range summary.Requests {
		deadline := "—"
		if r.DeadlineAt != nil {
			deadline = r.DeadlineAt.Format("02.01.06")
		}
		tasksCol := fmt.Sprintf("%d/%d", r.CompletedTasks, r.TotalTasks)
		hoursCol := fmt.Sprintf("%d/%d", r.CompletedHours, r.TotalHours)
		pdf.CellFormat(colW[0], 7, fmt.Sprintf("%d", r.RequestID), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[1], 7, truncate(r.Title, 28), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[2], 7, truncate(r.Contour, 12), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[3], 7, statusRu(reportStatusRu, r.Status), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[4], 7, tasksCol, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[5], 7, hoursCol, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[6], 7, deadline, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(6)
	pdf.SetFont(fontName, "", 9)
	pdf.CellFormat(0, 5, fmt.Sprintf("Сформировано: %s", summary.GeneratedAt.Format("02.01.2006 15:04")), "", 1, "R", false, 0, "")

	return pdfOutput(pdf)
}

func BuildReportPDF(report *ReportResponse) ([]byte, error) {
	fontName, pdf := pdfFont()
	pdf.AddPage()

	titleH := 10.0
	pdf.SetFont(fontName, "B", 16)
	pdf.CellFormat(0, titleH, "Отчёт по плану развёртывания", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont(fontName, "", 11)
	writeRow(pdf, fontName, "Заявка №", fmt.Sprintf("%d", report.RequestID))
	writeRow(pdf, fontName, "Название", report.Title)
	writeRow(pdf, fontName, "Контур", report.Contour)
	writeRow(pdf, fontName, "Статус", statusRu(reportStatusRu, report.Status))
	writeRow(pdf, fontName, "Создана", report.CreatedAt.Format("02.01.2006 15:04"))
	if report.DeadlineAt != nil {
		writeRow(pdf, fontName, "Дедлайн", report.DeadlineAt.Format("02.01.2006 15:04"))
	}
	pdf.Ln(4)

	pdf.SetFont(fontName, "B", 12)
	pdf.CellFormat(0, 8, "Задачи", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	colW := []float64{55, 22, 28, 75}
	headers := []string{"Работа", "Часы", "Статус", "Комментарий"}
	pdf.SetFont(fontName, "B", 10)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(colW[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont(fontName, "", 10)
	pdf.SetFillColor(255, 255, 255)
	for _, t := range report.Tasks {
		name := t.Name
		if name == "" {
			name = "—"
		}
		comment := t.CustomerComment
		if comment == "" {
			comment = "—"
		}
		rowH := 7.0
		if len(comment) > 40 {
			rowH = 14.0
		}
		pdf.CellFormat(colW[0], rowH, truncate(name, 35), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[1], rowH, fmt.Sprintf("%d", t.NormativeHours), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[2], rowH, statusRu(taskStatusRu, t.Status), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[3], rowH, truncate(comment, 45), "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(6)
	pdf.SetFont(fontName, "B", 11)
	pdf.CellFormat(0, 7, fmt.Sprintf("Задач выполнено: %d из %d", report.CompletedTasks, report.TotalTasks), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("Нормочасов: %d (выполнено %d)", report.TotalHours, report.CompletedHours), "", 1, "L", false, 0, "")
	pdf.SetFont(fontName, "", 9)
	pdf.Ln(4)
	pdf.CellFormat(0, 5, fmt.Sprintf("Сформировано: %s", time.Now().Format("02.01.2006 15:04")), "", 1, "R", false, 0, "")

	return pdfOutput(pdf)
}

func writeRow(pdf *gofpdf.Fpdf, font, label, value string) {
	pdf.SetFont(font, "B", 11)
	pdf.CellFormat(40, 7, label+":", "", 0, "L", false, 0, "")
	pdf.SetFont(font, "", 11)
	pdf.CellFormat(0, 7, value, "", 1, "L", false, 0, "")
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
