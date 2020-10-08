package classify

import (
	"encoding/csv"
	"github.com/pkg/errors"
	"os"
	"strconv"
)

func OutputResult(data [][]float32, outFile string, precision int) error {
	fout, err := os.Create(outFile)
	if err != nil {
		return errors.Wrap(err, "创建输出文件失败")
	}
	writer := csv.NewWriter(fout)
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
	_ = fout.Close()
	return nil
}
