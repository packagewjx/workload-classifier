package alitrace

import (
	"bufio"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/pkg/errors"
	"io"
	"log"
	"strconv"
	"strings"
)

// TODO 使用float数组而不是string
func imputeData(record []string) {
	recordIndex := func(fieldIndex, sectionIndex int) int {
		return sectionIndex*internal.NumSectionFields + fieldIndex
	}
	imputeFunc := func(start, end float64, fieldIndex, leftInclusive, rightInclusive int) {
		// +2 是因为需要保证左右有效值就是start和end，而不是第一个和最后一个NaN是start和end
		// data[leftInclusive-1]=start data[rightInclusive+1]=end
		k := (end - start) / float64(rightInclusive-leftInclusive+2)
		for i := leftInclusive; i <= rightInclusive; i++ {
			record[recordIndex(fieldIndex, i)] = fmt.Sprintf("%.2f", start+k*float64(i-(leftInclusive-1)))
		}
	}

	for i := 0; i < internal.NumSectionFields; i++ {
		invalidLeft := -1
		for j := 0; j < internal.NumSections; j++ {
			if record[recordIndex(i, j)] == "NaN" {
				if invalidLeft == -1 {
					invalidLeft = j
				}
			} else {
				if invalidLeft != -1 {
					startVal := 0.0
					if invalidLeft != 0 {
						startVal, _ = strconv.ParseFloat(record[recordIndex(i, invalidLeft-1)], 64)
					}
					endVal, _ := strconv.ParseFloat(record[recordIndex(i, j)], 64)

					imputeFunc(startVal, endVal, i, invalidLeft, j-1)
					invalidLeft = -1
				}
			}
		}

		// 检查最后的区间是否为NaN
		if invalidLeft != -1 {
			if invalidLeft == 0 {
				// 这种情况是整段数据都为NaN，暂时没有办法填充
			} else {
				startVal, _ := strconv.ParseFloat(record[recordIndex(i, invalidLeft-1)], 64)
				imputeFunc(startVal, 0, i, invalidLeft, internal.NumSections-1)
			}
		}
	}
}

func ImputeMissingValues(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	defer func() {
		_ = writer.Flush()
	}()

	var line string
	var err error
	lineCount := 0
	for line, err = reader.ReadString(internal.LineBreak); err == nil || (line != "" && err == io.EOF); line, err = reader.ReadString(internal.LineBreak) {
		lineCount++
		if strings.Contains(line, "NaN") {
			log.Printf("第%d行记录有NaN值，正在插值\n", lineCount)

			record := strings.Split(strings.TrimSpace(line), internal.Splitter)
			if len(record) < internal.NumSections*internal.NumSectionFields {
				return errors.New("文件记录格式不对")
			}
			startPos := len(record) - internal.NumSections*internal.NumSectionFields

			imputeData(record[startPos:])

			line = strings.Join(record, internal.Splitter) + string(internal.LineBreak)
		}
		n, err := writer.WriteString(line)
		if err != nil {
			return errors.Wrap(err, "输出文件错误")
		}
		if n != len(line) {
			return errors.New("输出不足")
		}
	}

	return nil
}
