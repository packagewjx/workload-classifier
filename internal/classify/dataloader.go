package classify

import (
	"encoding/csv"
	"github.com/pkg/errors"
	"io"
	"log"
	"math"
	"os"
	"strconv"
)

type DataFileLoader interface {
	Load(fileName string, removeColumn []int) ([][]float32, error)
}

type DataFormat string

const (
	CSV = DataFormat("csv")
)

func NewDataLoader(format DataFormat) DataFileLoader {
	switch format {
	case CSV:
		return &csvLoader{}
	default:
		return nil
	}
}

type csvLoader struct {
}

func (c *csvLoader) Load(fileName string, removeColumn []int) ([][]float32, error) {
	fin, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "打开csv文件出错")
	}
	reader := csv.NewReader(fin)

	data := make([][]float32, 0, 16)

	removeSet := make(map[int]struct{})
	for _, rc := range removeColumn {
		removeSet[rc] = struct{}{}
	}

	var record []string
	recordRead := 0
	for record, err = reader.Read(); err == nil; record, err = reader.Read() {
		recordRead++

		datum := make([]float32, 0, len(record))
		for i := 0; i < len(record); i++ {
			if _, ok := removeSet[i]; ok {
				continue
			}

			float, err := strconv.ParseFloat(record[i], 32)
			if err != nil || math.IsNaN(float) {
				log.Printf("第%d行第%d个数据有误，数据为[%v]", recordRead, i, record[i])
				datum = append(datum, 0)
			}
			datum = append(datum, float32(float))
		}

		data = append(data, datum)
	}

	if err != io.EOF {
		return nil, errors.Wrap(err, "读取数据出错")
	}

	return data, nil
}
