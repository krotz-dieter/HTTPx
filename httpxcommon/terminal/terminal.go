package terminal

import (
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func Println(m ...any) {
	pterm.Println(m...)
}

func PrintHeader(m ...any) {
	pterm.Println(pterm.FgCyan.Sprint(m...))
}

func PrintWarning(m ...any) {
	pterm.Warning.Println(m...)
}

func PrintSuccess(m ...any) {
	pterm.Success.Println(m...)
}

func PrintTreeListItems(items []string) {
	leveledList := pterm.LeveledList{}

	for _, item := range items {
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 0, Text: item})
	}

	root := putils.TreeFromLeveledList(leveledList)

	pterm.DefaultTree.WithRoot(root).Render()
}

func StartSpinner(m ...any) (any, error) {
	pSpinner, err := pterm.DefaultSpinner.Start(m...)
	if err != nil {
		return nil, err
	}
	return pSpinner, nil
}

func PrintTableWithHeaders(table [][]string) {
	pterm.DefaultTable.WithHasHeader().WithData(table).Render()
}

func PrintRedFg(text string) string {
	return pterm.FgRed.Sprint(text)
}

func PrintGreenFg(text string) string {
	return pterm.FgGreen.Sprint(text)
}
