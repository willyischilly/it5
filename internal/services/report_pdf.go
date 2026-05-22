package services

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

func BuildReportPDF(report *ReportResponse) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Deployment plan report")
	pdf.Ln(14)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 8, fmt.Sprintf("Request ID: %d", report.RequestID))
	pdf.Ln(8)
	pdf.Cell(40, 8, fmt.Sprintf("Title: %s", report.Title))
	pdf.Ln(8)
	pdf.Cell(40, 8, fmt.Sprintf("Contour: %s", report.Contour))
	pdf.Ln(8)
	pdf.Cell(40, 8, fmt.Sprintf("Created: %s", report.CreatedAt.Format("2006-01-02 15:04")))
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(80, 8, "Task")
	pdf.Cell(30, 8, "Hours")
	pdf.Cell(40, 8, "Status")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 11)
	for _, t := range report.Tasks {
		pdf.Cell(80, 7, t.Name)
		pdf.Cell(30, 7, fmt.Sprintf("%d", t.NormativeHours))
		pdf.Cell(40, 7, t.Status)
		pdf.Ln(8)
	}

	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(40, 8, fmt.Sprintf("Tasks: %d / %d completed", report.CompletedTasks, report.TotalTasks))
	pdf.Ln(8)
	pdf.Cell(40, 8, fmt.Sprintf("Hours: %d / %d completed", report.CompletedHours, report.TotalHours))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
