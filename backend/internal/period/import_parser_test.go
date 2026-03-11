package period

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParsePlanningPeriodWorkbook_AllowsDashValues(t *testing.T) {
	workbook := excelize.NewFile()
	sheet := workbook.GetSheetName(0)

	if err := workbook.SetSheetRow(sheet, "A1", &[]any{"Целевой индикатор", "ед. изм.", "2023", "2024"}); err != nil {
		t.Fatalf("failed to write header row: %v", err)
	}
	if err := workbook.SetSheetRow(sheet, "A2", &[]any{"Тестовый индикатор", "ед.", "-", "1200-1400"}); err != nil {
		t.Fatalf("failed to write data row: %v", err)
	}

	rows, skipped, err := parsePlanningPeriodWorkbook(workbook)
	if err != nil {
		t.Fatalf("parsePlanningPeriodWorkbook returned error: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("expected skipped=0, got %d", skipped)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	got2023 := rows[0].YearValues["2023"]
	if got2023 != "-" {
		t.Fatalf("expected 2023 value '-' , got '%s'", got2023)
	}

	got2024 := rows[0].YearValues["2024"]
	if got2024 != "1200-1400" {
		t.Fatalf("expected 2024 value '1200-1400', got '%s'", got2024)
	}
}

func TestValidateYearValues_AcceptsStringTargets(t *testing.T) {
	err := validateYearValues(map[string]string{
		"2023": "-",
		"2024": "1200-1400",
		"2025": "7",
	})
	if err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}
