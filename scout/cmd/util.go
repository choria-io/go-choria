package scoutcmd

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

func newMarkdownTable(hdr ...string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader(hdr)

	return table
}
