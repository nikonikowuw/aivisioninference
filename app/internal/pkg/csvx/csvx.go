// Package csvx 提供 CSV 导出辅助函数。
package csvx

import (
	"bytes"
	"encoding/csv"
	"strings"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Build 根据表头和数据行生成带 UTF-8 BOM 的 CSV 字节内容。
func Build(headers []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(utf8BOM)
	writer := csv.NewWriter(&buf)
	if err := writer.Write(sanitizeRow(headers)); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := writer.Write(sanitizeRow(row)); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sanitizeRow(row []string) []string {
	out := make([]string, len(row))
	for i, cell := range row {
		out[i] = EscapeFormula(cell)
	}
	return out
}

// EscapeFormula 防止 Excel 将用户可控字段按公式执行。
func EscapeFormula(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}
