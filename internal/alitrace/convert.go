package alitrace

import (
	"bufio"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"io"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
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

	return ProcessRawData(rawData), nil
}

func (u *containerWorkloadReader) readRawData() (map[string][]*internal.RawSectionData, error) {
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
	return containerSection, nil
}

func ProcessRawData(containerSection map[string][]*internal.RawSectionData) []*internal.ContainerWorkloadData {
	// 处理数据
	cDataArray := make([]*internal.ContainerWorkloadData, len(containerSection))
	idx := 0
	wg := sync.WaitGroup{}
	for cid, sections := range containerSection {
		wg.Add(1)
		go func(i int, containerId string, rawData []*internal.RawSectionData) {
			defer wg.Done()
			cData := &internal.ContainerWorkloadData{
				ContainerId: cid,
				Data:        make([]*internal.SectionData, len(rawData)),
			}

			for si, section := range rawData {
				stat := &internal.SectionData{}
				if len(section.Cpu) != 0 {
					stat.CpuAvg = section.CpuSum / float32(len(section.Cpu))
					stat.CpuMax = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)-1)
					stat.CpuMin = utils.GetSortedPositionValue(section.Cpu, 0)
					stat.CpuP50 = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)/2)
					stat.CpuP90 = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)*90/100)
					stat.CpuP99 = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)*99/100)
				} else {
					stat.CpuAvg = float32(math.NaN())
					stat.CpuMax = float32(math.NaN())
					stat.CpuMin = float32(math.NaN())
					stat.CpuP50 = float32(math.NaN())
					stat.CpuP90 = float32(math.NaN())
					stat.CpuP99 = float32(math.NaN())
				}
				if len(section.Mem) != 0 {
					stat.MemAvg = section.MemSum / float32(len(section.Mem))
					stat.MemMax = utils.GetSortedPositionValue(section.Mem, len(section.Mem)-1)
					stat.MemMin = utils.GetSortedPositionValue(section.Mem, 0)
					stat.MemP50 = utils.GetSortedPositionValue(section.Mem, len(section.Mem)/2)
					stat.MemP90 = utils.GetSortedPositionValue(section.Mem, len(section.Mem)*90/100)
					stat.MemP99 = utils.GetSortedPositionValue(section.Mem, len(section.Mem)*99/100)
				} else {
					stat.MemAvg = float32(math.NaN())
					stat.MemMax = float32(math.NaN())
					stat.MemMin = float32(math.NaN())
					stat.MemP50 = float32(math.NaN())
					stat.MemP90 = float32(math.NaN())
					stat.MemP99 = float32(math.NaN())
				}
				cData.Data[si] = stat
			}
			cDataArray[i] = cData
		}(idx, cid, sections)
		idx++
	}
	wg.Wait()
	return cDataArray
}
