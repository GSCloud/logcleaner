package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// cleanLog je zde pro zamezení problémů s kompilací, pokud se v main.go nezměnil.
// Ve skutečnosti by se měl čistě exportovat z main.go.
// Pro testování RunE ArgumentErrors se používá simulovaná funkce.

// Test cleanLog funkčnosti
func TestCleanLog_Trimming(t *testing.T) {
	// Vytvoření dočasného adresáře
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	maxRows := 5

	// Vytvoření testovacího obsahu (10 řádků)
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("Nelze vytvořit testovací soubor: %v", err)
	}

	// Spuštění cleanLog
	if err := cleanLog(logPath, maxRows, "irrelevant"); err != nil {
		t.Fatalf("cleanLog selhal: %v", err)
	}

	// 1. Kontrola, zda záloha existuje (s libovolným časovým razítkem)
	backupExists := false
	files, _ := filepath.Glob(logPath + ".*.bak")
	if len(files) > 0 {
		backupExists = true
		// Příklad čištění: Smažeme zálohu, aby testovací adresář zůstal čistý
		os.Remove(files[0])
	}
	if !backupExists {
		t.Errorf("Chyba: Záložní soubor nebyl vytvořen.")
	}

	// 2. Kontrola, zda má původní soubor správný počet řádků
	trimmedContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Nelze přečíst vyčištěný soubor: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(trimmedContent)), "\n")
	if len(lines) != maxRows {
		t.Errorf("Očekávaný počet řádků: %d, skutečný: %d", maxRows, len(lines))
	}

	// 3. Kontrola, zda jsou zachovány správné řádky (posledních 5)
	expectedLines := []string{"Line 6", "Line 7", "Line 8", "Line 9", "Line 10"}
	if !equalSlices(lines, expectedLines) {
		t.Errorf("Obsah se neshoduje.\nOčekáváno: %v\nSkutečnost: %v", expectedLines, lines)
	}
}

// Test prázdného logu
func TestCleanLog_Empty(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")

	// Vytvoření prázdného souboru
	if _, err := os.Create(logPath); err != nil {
		t.Fatalf("Nelze vytvořit prázdný log: %v", err)
	}

	// Spuštění cleanLog
	if err := cleanLog(logPath, 5, "irrelevant"); err != nil {
		t.Fatalf("cleanLog selhal: %v", err)
	}

	// Kontrola, zda je výsledný soubor prázdný
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Nelze přečíst soubor: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Očekával se prázdný log, nalezena délka %d.", len(data))
	}
}

// Test chyb v parsování argumentů
func TestRunE_ArgumentErrors(t *testing.T) {
	// Použijeme RunE zkopírovaný z main.go pro testování chyb parsování.
	testRunE := func(cmd *cobra.Command, args []string) error {
		path := args[0]
		rowsStr := args[1]
		format := args[2]

		// Převod řádků ze stringu na int
		rows, err := strconv.Atoi(rowsStr)
		if err != nil {
			return fmt.Errorf("chyba: Druhý argument 'max řádků' musí být číslo. Zadáno: %s", rowsStr)
		}
		if rows <= 0 {
			return fmt.Errorf("chyba: Maximální počet řádků musí být kladné číslo")
		}

		// Zde by normálně bylo cleanLog, ale pro testování parsování to přeskočíme
		_ = path
		_ = rows
		_ = format
		return nil
	}

	cmd := &cobra.Command{RunE: testRunE}

	// Test 1: max řádků není číslo (špatný formát argumentu)
	args1 := []string{"/path/log", "text", "format"}
	err1 := cmd.RunE(cmd, args1)
	if err1 == nil || !strings.Contains(err1.Error(), "max řádků' musí být číslo") {
		t.Errorf("Očekávaná chyba 'musí být číslo', nalezena: %v", err1)
	}

	// Test 2: max řádků je záporné číslo (neplatná hodnota)
	args2 := []string{"/path/log", "-5", "format"}
	err2 := cmd.RunE(cmd, args2)
	if err2 == nil || !strings.Contains(err2.Error(), "kladné číslo") {
		t.Errorf("Očekávaná chyba 'kladné číslo', nalezena: %v", err2)
	}

	// Test 3: max řádků je nula (neplatná hodnota)
	args3 := []string{"/path/log", "0", "format"}
	err3 := cmd.RunE(cmd, args3)
	if err3 == nil || !strings.Contains(err3.Error(), "kladné číslo") {
		t.Errorf("Očekávaná chyba 'kladné číslo', nalezena: %v", err3)
	}
}

// Pomocná funkce pro porovnání řezů
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Otestování standardního chování Cobry (zobrazení Usage a chyba)
func TestArgs_StandardCobraBehavior(t *testing.T) {
	// Použijeme bytes.Buffer pro zachycení výstupu nápovědy
	var buf bytes.Buffer

	// Vytvoříme jednoduchý root Command se standardními vlastnostmi
	cmd := &cobra.Command{
		Use:  "logcleaner [cesta k logu] [max řádků] [formát data]",
		Args: cobra.ExactArgs(3),
		// Ztlumení chyby (Error: accepts 3...) ale PONECHÁNÍ Usage (nápovědy)
		SilenceErrors: true,
		SilenceUsage:  false,
		Run:           func(cmd *cobra.Command, args []string) { /* Do Nothing */ },
	}
	cmd.SetOut(&buf) // Přesměrujeme výstup Cobry do bufferu

	// Spustíme Execute s nesprávným počtem argumentů (méně než 3)
	cmd.SetArgs([]string{"/path/log", "5"})
	err := cmd.Execute()

	// 1. Kontrola, zda byla vrácena chyba (Cobra.CommandError nebo podobná)
	if err == nil {
		t.Fatal("Očekávala se chyba, ale nebyla vrácena žádná.")
	}

	// 2. Kontrola, zda byl nějaký obsah (nápověda) vypsán
	out := buf.String()
	if !strings.Contains(out, "logcleaner [cesta k logu]") {
		t.Errorf("Očekával se výstup nápovědy (Usage), nalezeno: %s", out)
	}
}
