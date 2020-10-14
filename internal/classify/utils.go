package classify

import (
	"encoding/csv"
	"github.com/pkg/errors"
	"io"
	"strconv"
)

func OutputResult(data [][]float32, output io.Writer, precision int) error {
	writer := csv.NewWriter(output)
	for _, datum := range data {
		record := make([]string, len(datum))
		for i, f := range datum {
			record[i] = strconv.FormatFloat(float64(f), 'f', precision, 32)
		}
		err := writer.Write(record)
		if err != nil {
			return errors.Wrap(err, "写入数据错误")
		}
	}

	writer.Flush()
	return nil
}
