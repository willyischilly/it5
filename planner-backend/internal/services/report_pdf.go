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
	sumAligns := []string{"C", "L", "C", "C", "C", "C", "C"}
	pdf.SetFont(fontName, "B", 9)
	pdfDrawTableRow(pdf, colW, sumAligns, headers, 6, true)

	pdf.SetFont(fontName, "", 9)
	for _, r := range summary.Requests {
		deadline := "—"
		if r.DeadlineAt != nil {
			deadline = r.DeadlineAt.Format("02.01.06")
		}
		cells := []string{
			fmt.Sprintf("%d", r.RequestID),
			r.Title,
			r.Contour,
			statusRu(reportStatusRu, r.Status),
			fmt.Sprintf("%d/%d", r.CompletedTasks, r.TotalTasks),
			fmt.Sprintf("%d/%d", r.CompletedHours, r.TotalHours),
			deadline,
		}
		pdfDrawTableRow(pdf, colW, sumAligns, cells, 6, false)
	}

	if len(summary.Tasks) > 0 {
		pdf.Ln(6)
		pdf.SetFont(fontName, "B", 12)
		pdf.CellFormat(0, 8, "Задачи", "", 1, "L", false, 0, "")
		pdf.Ln(2)

		taskColW := []float64{12, 30, 34, 34, 18, 12}
		taskAligns := []string{"C", "L", "L", "L", "C", "C"}
		taskHeaders := []string{"№", "Заявка", "Работа", "Исполнитель", "Статус", "Часы"}
		pdf.SetFont(fontName, "B", 9)
		pdfDrawTableRow(pdf, taskColW, taskAligns, taskHeaders, 6, true)

		pdf.SetFont(fontName, "", 9)
		for _, t := range summary.Tasks {
			workName := t.WorkName
			if workName == "" {
				workName = "—"
			}
			executor := t.ExecutorFullName
			if executor == "" {
				executor = "—"
			}
			taskCells := []string{
				fmt.Sprintf("%d", t.RequestID),
				t.RequestTitle,
				workName,
				executor,
				statusRu(taskStatusRu, t.Status),
				fmt.Sprintf("%d", t.NormativeHours),
			}
			pdfDrawTableRow(pdf, taskColW, taskAligns, taskCells, 6, false)
		}
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

	const labelW = 40.0
	const valueW = 138.0
	const lineH = 6.0

	pdf.SetFont(fontName, "", 11)
	writeRow(pdf, fontName, "Заявка №", fmt.Sprintf("%d", report.RequestID))
	writeRowWrap(pdf, fontName, "Название", report.Title, labelW, valueW, lineH)
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

	colW := []float64{28, 30, 32, 12, 18, 46}
	aligns := []string{"L", "L", "L", "C", "C", "L"}
	headers := []string{"Работа", "Описание", "Исполнитель", "Часы", "Статус", "Комментарий"}
	pdf.SetFont(fontName, "B", 9)
	pdfDrawTableRow(pdf, colW, aligns, headers, lineH, true)

	pdf.SetFont(fontName, "", 9)
	for _, t := range report.Tasks {
		name := t.Name
		if name == "" {
			name = "—"
		}
		desc := t.Description
		executor := t.ExecutorFullName
		if executor == "" {
			executor = "—"
		}
		comment := t.CustomerComment
		cells := []string{
			name,
			desc,
			executor,
			fmt.Sprintf("%d", t.NormativeHours),
			statusRu(taskStatusRu, t.Status),
			comment,
		}
		pdfDrawTableRow(pdf, colW, aligns, cells, lineH, false)
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

