package alitrace

import (
	"bufio"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/packagewjx/workload-classifier/internal/datasource"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
)

const StatusStarted = "started"

type ContainerWorkloadReader interface {
	Read() ([]*internal.ContainerWorkloadData, error)
}

func NewContainerWorkloadReader(in io.Reader, meta map[string][]*ContainerMeta) ContainerWorkloadReader {
	return &containerWorkloadReader{
		reader: bufio.NewReader(in),
		meta:   meta,
	}
}

type containerWorkloadReader struct {
	reader    *bufio.Reader
	meta      map[string][]*ContainerMeta
	ReadCount uint64
}

func (u *containerWorkloadReader) Read() ([]*internal.ContainerWorkloadData, error) {
	rawData, err := u.readRawData()
	if err != nil {
		return nil, err
	}

	return datasource.ConvertAllRawData(rawData), nil
}

func (u *containerWorkloadReader) readRawData() ([]*internal.ContainerRawData, error) {
	containerSection := make(map[string][]*internal.RawSectionData)

	// 读取输入
	var line string
	var err error
	lineNum := 0
	for line, err = u.reader.ReadString(internal.LineBreak); err == nil || (line != "" && err == io.EOF); line, err = u.reader.ReadString(internal.LineBreak) {
		lineNum++
		u.ReadCount += uint64(len(line))
		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		record := strings.Split(line, internal.Splitter)

		cid := record[0]
		timestamp, _ := strconv.ParseInt(record[2], 10, 64)
		statuses := u.meta[cid]
		// statusIdx应该是第一个大于等于timestamp的记录。这里要的应该是前一条记录，因为前一条记录的元数据信息才是本时间点的元数据
		statusIdx := sort.Search(len(statuses), func(i int) bool {
			return statuses[i].Timestamp >= timestamp
		})
		if len(statuses) == 0 {
			log.Printf("没有containerID为'%s'的元数据记录，行号%d\n", cid, lineNum)
			continue
		} else if statusIdx == 0 {
			log.Printf("不应该找到位置为0的记录，containerID: %s，行号：%d\n", cid, lineNum)
		} else {
			statusIdx--
		}

		if statuses[statusIdx].Status != StatusStarted {
			log.Printf("找到的状态不是开始，containerId：%s，行号%d\n", cid, lineNum)
		}

		// statuses留作后用

		sectionIndex := timestamp % internal.DayLength / internal.SectionLength
		cpuUtil, _ := strconv.ParseFloat(record[3], 32)
		memUtil, _ := strconv.ParseFloat(record[4], 32)
		sections, ok := containerSection[cid]
		if !ok {
			sections = make([]*internal.RawSectionData, internal.NumSections)
			for i := 0; i < len(sections); i++ {
				sections[i] = &internal.RawSectionData{
					Cpu: make([]float32, 0, 128),
					Mem: make([]float32, 0, 128),
				}
			}
			containerSection[cid] = sections
		}

		cpu := cpuUtil
		mem := memUtil

		sect := sections[sectionIndex]
		sect.Cpu = append(sect.Cpu, float32(cpu))
		sect.Mem = append(sect.Mem, float32(mem))
		sect.CpuSum += float32(cpu)
		sect.MemSum += float32(mem)
	}

	result := make([]*internal.ContainerRawData, 0, len(containerSection))
	for id, data := range containerSection {
		result = append(result, &internal.ContainerRawData{
			ContainerId: id,
			Data:        data,
		})
	}

	return result, nil
}
