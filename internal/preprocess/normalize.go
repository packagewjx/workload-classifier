package preprocess

import (
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
)

func Normalize() Preprocessor {
	return &normalize{}
}

type normalize struct {
}

func (n normalize) Preprocess(workload *core.ContainerWorkloadData) {
	data := workload.Data
	typ := reflect.TypeOf(core.SectionData{})
	dataVal := make([]reflect.Value, len(data))
	for i, datum := range data {
		dataVal[i] = reflect.ValueOf(datum)
	}

	// 寻找CPU和Mem的最大值
	maxCpu := float32(0)
	maxMem := float32(0)
	for i := 0; i < len(data); i++ {
		if maxCpu < data[i].CpuMax {
			maxCpu = data[i].CpuMax
		}
		if maxMem < data[i].MemMax {
			maxMem = data[i].MemMax
		}
	}

	for i := 0; i < typ.NumField(); i++ {
		fieldName := typ.Field(i).Name
		max := float32(0)
		if -1 != strings.Index(fieldName, "Cpu") {
			max = maxCpu
		} else if -1 != strings.Index(fieldName, "Mem") {
			max = maxMem
		}
		if max == 0 {
			continue
		}

		for _, datum := range dataVal {
			field := datum.Elem().FieldByName(fieldName)
			field.SetFloat(field.Float() / float64(max))
		}
	}
}

func NormalizeSection(in io.Reader, out io.Writer) error {
	log.Println("正在读取数据")
	records, err := csv.NewReader(in).ReadAll()
	if err != nil {
		return errors.Wrap(err, "读取数据失败")
	}

	cDataArray := make([]*core.ContainerWorkloadData, len(records))

	log.Println("读取完毕，正在转换数据")
	wg := sync.WaitGroup{}
	errCh := make(chan error)
	doneCh := make(chan struct{})
	normalize := Normalize()

	for i, record := range records {
		if len(record) < core.NumSectionFields*core.NumSections {
			return fmt.Errorf("第%d行数据有问题，数据长度不足%d",
				i, core.NumSectionFields*core.NumSections)
		}
		wg.Add(1)
		go func(idx int, record []string) {
			defer wg.Done()

			cData, err := utils.RecordToContainerWorkloadData(record)
			if err != nil {
				errCh <- errors.Wrap(err, fmt.Sprintf("解析第%d行数据失败", idx))
				return
			}
			normalize.Preprocess(cData)
			cDataArray[idx] = cData
		}(i, record)
	}

	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	select {
	case <-doneCh:
		break
	case err := <-errCh:
		// fail fast
		return err
	}

	log.Println("转换完毕，正在写出数据")
	return utils.WriteContainerWorkloadData(out, cDataArray)
}
