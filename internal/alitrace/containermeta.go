package alitrace

import (
	"encoding/csv"
	"github.com/pkg/errors"
	"io"
	"log"
	"strconv"
)

type ContainerMeta struct {
	ContainerId string
	MachineId   string
	Timestamp   int64
	AppDu       string
	Status      string
	CpuRequest  int
	CpuLimit    int
	MemSize     float32
}

func LoadContainerMeta(in io.Reader) (map[string][]*ContainerMeta, error) {
	reader := csv.NewReader(in)

	containerMeta := make(map[string][]*ContainerMeta)
	var record []string
	var err error
	cnt := 0
	for record, err = reader.Read(); err == nil; record, err = reader.Read() {
		cnt++
		cid := record[0]
		s, ok := containerMeta[cid]
		if !ok {
			s = make([]*ContainerMeta, 0, 10)
		}
		timestamp, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			log.Printf("第%d条记录的timestamp格式错误，记录为%s，错误为%v\n", cnt, record[2], err)
		}
		cpuRequest, err := strconv.ParseInt(record[5], 10, 64)
		if err != nil {
			log.Printf("第%d条记录的cpu request格式错误，记录为%s，错误为%v\n", cnt, record[5], err)
		}
		cpuLimit, err := strconv.ParseInt(record[6], 10, 64)
		if err != nil {
			log.Printf("第%d条记录的cpu limit格式错误，记录为%s，错误为%v\n", cnt, record[6], err)
		}
		memSize, err := strconv.ParseFloat(record[7], 64)
		if err != nil {
			log.Printf("第%d条记录的mem size格式错误，记录为%s，错误为%v\n", cnt, record[7], err)
		}

		s = append(s, &ContainerMeta{
			ContainerId: cid,
			MachineId:   record[1],
			Timestamp:   timestamp,
			AppDu:       record[3],
			Status:      record[4],
			CpuRequest:  int(cpuRequest),
			CpuLimit:    int(cpuLimit),
			MemSize:     float32(memSize),
		})
		containerMeta[cid] = s
	}
	if err != io.EOF {
		return nil, errors.Wrap(err, "读取元数据错误")
	}

	return containerMeta, nil
}
