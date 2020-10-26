package preprocess

import (
	"bufio"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"io"
	"log"
	"math"
	"reflect"
	"strings"
)

func Impute() Preprocessor {
	return &imputePreProcessor{}
}

type imputePreProcessor struct {
}

func (i imputePreProcessor) Preprocess(workload *core.ContainerWorkloadData) {
	for fi := 0; fi < core.NumSectionFields; fi++ {
		invalidLeft := -1
		for si := 0; si < len(workload.Data); si++ {
			f := reflect.ValueOf(workload.Data[si]).Elem().Field(fi).Float()
			if math.IsNaN(f) {
				if invalidLeft == -1 {
					invalidLeft = si
				}
			} else {
				if invalidLeft != -1 {
					startVal := 0.0
					if invalidLeft != 0 {
						startVal = reflect.ValueOf(workload.Data[invalidLeft-1]).Elem().Field(fi).Float()
					}
					endVal := f

					// 线性填充
					k := (endVal - startVal) / float64(si-invalidLeft+1)
					for i := invalidLeft; i < si; i++ {
						reflect.ValueOf(workload.Data[i]).Elem().Field(fi).SetFloat(startVal + k*float64(i-(invalidLeft-1)))
					}

					invalidLeft = -1
				}
			}
		}

		if invalidLeft != -1 {
			if invalidLeft == 0 {
				// 整段都是NaN
			} else {
				startVal := reflect.ValueOf(workload.Data[invalidLeft-1]).Elem().Field(fi).Float()
				k := (-startVal) / float64(len(workload.Data)-invalidLeft+1)
				for i := invalidLeft; i < len(workload.Data); i++ {
					reflect.ValueOf(workload.Data[i]).Elem().Field(fi).SetFloat(startVal + k*float64(i-(invalidLeft-1)))
				}
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
	impute := Impute()
	for line, err = reader.ReadString(core.LineBreak); err == nil || (line != "" && err == io.EOF); line, err = reader.ReadString(core.LineBreak) {
		lineCount++
		if strings.Contains(line, "NaN") {
			log.Printf("第%d行记录有NaN值，正在插值\n", lineCount)

			line = strings.TrimSpace(line)
			data, err := utils.RecordToContainerWorkloadData(strings.Split(line, core.Splitter))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("第%d行数据错误", lineCount))
			}
			impute.Preprocess(data)

			record := utils.WorkloadDataToStringRecord(data)
			line = strings.Join(record, core.Splitter) + string(core.LineBreak)
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
